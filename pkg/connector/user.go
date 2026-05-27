package connector

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const memberIdProfileKey = "member_id"

type UserResourceType struct {
	resourceType *v2.ResourceType
	client       *cloudflare.API
	accountId    string
}

func (o *UserResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func userResource(member cloudflare.AccountMember) (*v2.Resource, error) {
	user := member.User
	firstName := user.FirstName
	lastName := user.LastName
	profile := map[string]interface{}{
		"login":            user.Email,
		"first_name":       firstName,
		"last_name":        lastName,
		"email":            user.Email,
		memberIdProfileKey: member.ID,
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(v2.UserTrait_Status_STATUS_UNSPECIFIED),
		rs.WithUserLogin(user.Email),
		rs.WithEmail(user.Email, true),
	}

	displayName := user.FirstName
	if user.FirstName == "" {
		displayName = user.Email
	}

	resource, err := rs.NewUserResource(displayName, resourceTypeUser, user.ID, userTraits)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (o *UserResourceType) List(ctx context.Context, _ *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	page, err := convertPageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("Cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	users, resp, err := o.client.AccountMembers(ctx, o.accountId, pageOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("cloudflare: could not retrieve users: %w", err)
	}

	nextPage := convertNextPageToken(resp.Page, len(users))
	rv := make([]*v2.Resource, 0, len(users))
	for _, user := range users {
		userResource, err := userResource(user)
		if err != nil {
			return nil, nil, err
		}
		rv = append(rv, userResource)
	}

	return rv, &rs.SyncOpResults{NextPageToken: nextPage}, nil
}

func (o *UserResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *UserResourceType) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *UserResourceType) CreateAccountCapabilityDetails(_ context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

// CreateAccount invites a user to join the Cloudflare account.
// Cloudflare uses an invitation model — the user receives an email and must accept before gaining access.
// The profile may include a "roles" field ([]interface{} of role ID strings) to assign initial roles.
// At least one role ID is required by the Cloudflare API.
func (o *UserResourceType) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	_ *v2.LocalCredentialOptions,
) (connectorbuilder.CreateAccountResponse, []*v2.PlaintextData, annotations.Annotations, error) {
	email, firstName, lastName, err := getAccountInfo(accountInfo)
	if err != nil {
		return nil, nil, nil, err
	}

	roleIDs := getRoleIDsFromProfile(accountInfo)

	member, err := o.client.CreateAccountMember(ctx, cloudflare.AccountIdentifier(o.accountId), cloudflare.CreateAccountMemberParams{
		EmailAddress: email,
		Roles:        roleIDs,
		Status:       "pending",
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-cloudflare: failed to invite account member: %w", err)
	}

	// Overlay the operator-supplied name since the invited user's Cloudflare profile
	// may be empty until they accept; this gives C1 a meaningful display name immediately.
	if firstName != "" {
		member.User.FirstName = firstName
	}
	if lastName != "" {
		member.User.LastName = lastName
	}

	resource, err := userResource(member)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-cloudflare: failed to build user resource after invite: %w", err)
	}

	return &v2.CreateAccountResponse_ActionRequiredResult{
		Resource: resource,
		Message:  "A Cloudflare account invitation has been sent. The user must accept the invitation before gaining access.",
	}, nil, nil, nil
}

// Delete removes a user from the Cloudflare account.
// The resource ID is the Cloudflare user UUID; the member ID is resolved via API lookup.
func (o *UserResourceType) Delete(ctx context.Context, resourceId *v2.ResourceId, _ *v2.ResourceId) (annotations.Annotations, error) {
	if resourceId.ResourceType != resourceTypeUser.Id {
		return nil, fmt.Errorf("baton-cloudflare: invalid resource type for delete: %s", resourceId.ResourceType)
	}

	memberID, err := findMemberIDByUserID(ctx, o.client, o.accountId, resourceId.Resource)
	if err != nil {
		return nil, err
	}

	err = o.client.DeleteAccountMember(ctx, o.accountId, memberID)
	if err != nil {
		var notFound cloudflare.NotFoundError
		if isCloudflareNotFound(err, &notFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("baton-cloudflare: failed to remove account member: %w", err)
	}

	return nil, nil
}

func userBuilder(client *cloudflare.API, accountId string) *UserResourceType {
	return &UserResourceType{
		resourceType: resourceTypeUser,
		client:       client,
		accountId:    accountId,
	}
}
