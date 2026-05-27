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

// getPrimaryEmail returns the primary email address from AccountInfo, or the first
// available email if no primary is marked.
func getPrimaryEmail(accountInfo *v2.AccountInfo) string {
	emails := accountInfo.GetEmails()
	for _, e := range emails {
		if e.GetIsPrimary() {
			return e.GetAddress()
		}
	}
	if len(emails) > 0 {
		return emails[0].GetAddress()
	}
	return ""
}

// getRoleIDsFromProfile extracts a list of Cloudflare role IDs from the account info profile.
// The profile field "roles" must be a list of role ID strings.
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
	rolesList, ok := rolesVal.([]interface{})
	if !ok {
		return nil
	}
	var roleIDs []string
	for _, r := range rolesList {
		if roleID, ok := r.(string); ok && roleID != "" {
			roleIDs = append(roleIDs, roleID)
		}
	}
	return roleIDs
}

// isCloudflareNotFound checks whether an error is a Cloudflare NotFoundError.
func isCloudflareNotFound(err error, target *cloudflare.NotFoundError) bool {
	return errors.As(err, target)
}
