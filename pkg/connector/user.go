package connector

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/sdk"
)

type UserResourceType struct {
	resourceType *v2.ResourceType
	api          *cloudflare.API
	accountId    string
}

func (o *UserResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func (o *UserResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	page, err := convertPageToken(pt.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	users, resp, err := o.api.AccountMembers(ctx, o.accountId, pageOpts)
	if err != nil {
		return nil, "", nil, fmt.Errorf("cloudflare: could not retrieve users: %w", err)
	}

	nextPage := convertNextPageToken(resp.Page, len(users))

	rv := make([]*v2.Resource, 0, len(users))
	for _, user := range users {
		annos := &v2.V1Identifier{
			Id: user.User.ID,
		}
		profile := userProfile(ctx, user)
		userResource, err := sdk.NewUserResource(user.User.Email, resourceTypeUser, nil, user.User.ID, user.User.Email, profile, annos)
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

func userBuilder(api *cloudflare.API, accountId string) *UserResourceType {
	return &UserResourceType{
		resourceType: resourceTypeUser,
		api:          api,
		accountId:    accountId,
	}
}

func userProfile(ctx context.Context, user cloudflare.AccountMember) map[string]interface{} {
	profile := make(map[string]interface{})
	profile["first_name"] = user.User.FirstName
	profile["last_name"] = user.User.LastName
	profile["user_id"] = user.User.ID

	return profile
}
