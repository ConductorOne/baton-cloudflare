package connector

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

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
		"login":      user.Email,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      user.Email,
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

func (o *UserResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	page, err := convertPageToken(pt.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	users, resp, err := o.client.AccountMembers(ctx, o.accountId, pageOpts)
	if err != nil {
		return nil, "", nil, fmt.Errorf("cloudflare: could not retrieve users: %w", err)
	}

	nextPage := convertNextPageToken(resp.Page, len(users))
	rv := make([]*v2.Resource, 0, len(users))
	for _, user := range users {
		userResource, err := userResource(user)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, userResource)
	}

	return rv, nextPage, nil, nil
}

func (o *UserResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *UserResourceType) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func userBuilder(client *cloudflare.API, accountId string) *UserResourceType {
	return &UserResourceType{
		resourceType: resourceTypeUser,
		client:       client,
		accountId:    accountId,
	}
}
