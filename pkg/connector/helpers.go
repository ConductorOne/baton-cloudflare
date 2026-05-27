package connector

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

const (
	MembershipEntitlementIDTemplate = "membership:%s"
	V1GrantIDTemplate               = "grant:%s:%s"
)

type CloudflareError struct {
	ErrorMessage     string                   `json:"error"`
	ErrorDescription string                   `json:"error_description"`
	ErrorCode        int                      `json:"errorCode,omitempty"`
	ErrorSummary     string                   `json:"errorSummary,omitempty" toml:"error_description"`
	ErrorLink        string                   `json:"errorLink,omitempty"`
	ErrorId          string                   `json:"errorId,omitempty"`
	ErrorCauses      []map[string]interface{} `json:"errorCauses,omitempty"`
}

func (b *CloudflareError) Error() string {
	return b.ErrorMessage
}

func capabilityPermissions(perms ...string) *v2.CapabilityPermissions {
	cp := &v2.CapabilityPermissions{}
	for _, p := range perms {
		cp.Permissions = append(cp.Permissions, &v2.CapabilityPermission{Permission: p})
	}
	return cp
}

func v1AnnotationsForResourceType(resourceTypeID string) annotations.Annotations {
	annos := annotations.Annotations{}
	annos.Update(&v2.V1Identifier{
		Id: resourceTypeID,
	})
	return annos
}

func v1AnnotationsWithPermissions(resourceTypeID string, perms *v2.CapabilityPermissions) annotations.Annotations {
	annos := v1AnnotationsForResourceType(resourceTypeID)
	annos.Update(perms)
	return annos
}

func V1MembershipEntitlementID(resourceID string) string {
	return fmt.Sprintf(MembershipEntitlementIDTemplate, resourceID)
}

func V1GrantID(entitlementID string, userID string) string {
	return fmt.Sprintf(V1GrantIDTemplate, entitlementID, userID)
}

func convertPageToken(token string) (int, error) {
	if token == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(token)
	return page, err
}

func convertNextPageToken(token int, responseLength int) string {
	if responseLength == 0 {
		return ""
	}
	strToken := strconv.FormatInt(int64(token)+1, 10)
	return strToken
}

func getError(err error) error {
	var bitbucketErr *CloudflareError
	if err == nil {
		return nil
	}

	if errors.As(err, &bitbucketErr) {
		return fmt.Errorf("%s %s", bitbucketErr.Error(), bitbucketErr.ErrorSummary)
	}

	return err
}

func WithAuthorizationBearerHeader(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", "Bearer "+token)
}

// findMemberIDByUserID looks up the Cloudflare membership ID for a given user UUID.
// The resource ID stored in baton is the user UUID (member.User.ID), but Cloudflare's
// delete and update APIs require the membership ID (member.ID).
func findMemberIDByUserID(ctx context.Context, client *cloudflare.API, accountID, userID string) (string, error) {
	perPage := 50
	page := 1
	processed := 0

	for {
		members, resp, err := client.AccountMembers(ctx, accountID, cloudflare.PaginationOptions{
			Page:    page,
			PerPage: perPage,
		})
		if err != nil {
			return "", fmt.Errorf("baton-cloudflare: failed to list account members: %w", err)
		}

		for _, m := range members {
			if m.User.ID == userID {
				return m.ID, nil
			}
		}

		processed += len(members)
		if processed >= resp.Total {
			break
		}
		page++
	}

	return "", fmt.Errorf("baton-cloudflare: account member not found for user ID %s", userID)
}

// getAccountInfo extracts the primary email and optional first/last name from AccountInfo.
// Email comes from the C1 user's primary email; name fields come from the provisioning profile.
func getAccountInfo(accountInfo *v2.AccountInfo) (string, string, string, error) {
	email := ""
	for _, e := range accountInfo.GetEmails() {
		if e.GetIsPrimary() {
			email = e.GetAddress()
			break
		}
	}
	if email == "" && len(accountInfo.GetEmails()) > 0 {
		email = accountInfo.GetEmails()[0].GetAddress()
	}
	if email == "" {
		return "", "", "", fmt.Errorf("baton-cloudflare: primary email is required to invite an account member")
	}

	profile := map[string]interface{}{}
	if accountInfo.GetProfile() != nil {
		profile = accountInfo.GetProfile().AsMap()
	}
	firstName, _ := profile["first_name"].(string)
	lastName, _ := profile["last_name"].(string)
	return email, firstName, lastName, nil
}

// getRoleIDsFromProfile extracts a list of Cloudflare role IDs from the account info profile.
// The profile field "roles" may arrive as []interface{} (StringListField) or a single string.
func getRoleIDsFromProfile(accountInfo *v2.AccountInfo) []string {
	profile := accountInfo.GetProfile()
	if profile == nil {
		return nil
	}
	profileMap := profile.AsMap()
	rolesVal, ok := profileMap["roles"]
	if !ok {
		return nil
	}

	var roleIDs []string
	switch v := rolesVal.(type) {
	case []interface{}:
		for _, r := range v {
			if roleID, ok := r.(string); ok && roleID != "" {
				roleIDs = append(roleIDs, roleID)
			}
		}
	case string:
		if v != "" {
			roleIDs = append(roleIDs, v)
		}
	}
	return roleIDs
}

// isCloudflareNotFound checks whether an error is a Cloudflare NotFoundError.
func isCloudflareNotFound(err error, target *cloudflare.NotFoundError) bool {
	return errors.As(err, target)
}
