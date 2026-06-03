package connector

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type InvitationResourceType struct {
	resourceType *v2.ResourceType
	client       *cloudflare.API
	accountId    string
}

func (o *InvitationResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func invitationResource(member cloudflare.AccountMember) (*v2.Resource, error) {
	email := member.User.Email
	profile := map[string]interface{}{
		"email":  email,
		"status": member.Status,
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(v2.UserTrait_Status_STATUS_UNSPECIFIED),
		rs.WithUserLogin(email),
		rs.WithEmail(email, true),
	}

	// member.ID (the membership UUID) is used as the resource ID because member.User.ID
	// is empty for pending invitations until the user accepts and gets a Cloudflare UUID.
	resource, err := rs.NewUserResource(email, resourceTypeInvitation, member.ID, userTraits)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (o *InvitationResourceType) List(ctx context.Context, _ *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	page, err := convertPageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	members, resp, err := o.client.AccountMembers(ctx, o.accountId, pageOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-cloudflare: could not retrieve account members: %w", err)
	}

	nextPage := convertNextPageToken(resp.Page, len(members))
	rv := make([]*v2.Resource, 0)
	for _, member := range members {
		// Only capture pending invitations — accepted members are handled by the user resource type.
		if member.User.ID != "" {
			continue
		}
		resource, err := invitationResource(member)
		if err != nil {
			return nil, nil, err
		}
		rv = append(rv, resource)
	}

	return rv, &rs.SyncOpResults{NextPageToken: nextPage}, nil
}

func (o *InvitationResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *InvitationResourceType) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// Delete cancels a pending invitation.
// The resource ID is the membership UUID (member.ID), which is what the Cloudflare delete API requires.
func (o *InvitationResourceType) Delete(ctx context.Context, resourceId *v2.ResourceId, _ *v2.ResourceId) (annotations.Annotations, error) {
	if resourceId.ResourceType != resourceTypeInvitation.Id {
		return nil, fmt.Errorf("baton-cloudflare: invalid resource type for delete: %s", resourceId.ResourceType)
	}

	err := o.client.DeleteAccountMember(ctx, o.accountId, resourceId.Resource)
	if err != nil {
		var notFound *cloudflare.NotFoundError
		if errors.As(err, &notFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("baton-cloudflare: failed to cancel invitation: %w", err)
	}

	return nil, nil
}

func invitationBuilder(client *cloudflare.API, accountId string) *InvitationResourceType {
	return &InvitationResourceType{
		resourceType: resourceTypeInvitation,
		client:       client,
		accountId:    accountId,
	}
}
