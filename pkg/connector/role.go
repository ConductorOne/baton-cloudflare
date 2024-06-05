package connector

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/sdk"
)

const (
	roleMemberEntitlement = "member"
	// The list custom roles endpoint does not return the super admin role, so we are manually adding it with Cloudflares super admin role ID.
	SuperAdminRoleId = "33666b9c79b9a5273fc7344ff42f953d"
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	api          *cloudflare.API
	accountId    string
}

func (o *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func (o *roleResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	roles, err := o.api.AccountRoles(ctx, o.accountId)
	if err != nil {
		return nil, "", nil, err
	}
	rv := make([]*v2.Resource, 0, len(roles))
	for _, r := range roles {
		annos := &v2.V1Identifier{
			Id: r.ID,
		}
		profile := roleProfile(ctx, r)
		roleResource, err := sdk.NewRoleResource(r.Name, resourceTypeRole, nil, r.ID, profile, annos)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, roleResource)
	}
	annos := &v2.V1Identifier{
		Id: SuperAdminRoleId,
	}
	adminRoleResource, err := sdk.NewRoleResource("Super Administrator - All Privileges", resourceTypeRole, nil, SuperAdminRoleId, nil, annos)
	if err != nil {
		return nil, "", nil, err
	}
	rv = append(rv, adminRoleResource)
	return rv, "", nil, nil
}

func (o *roleResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var annos annotations.Annotations
	annos.Update(&v2.V1Identifier{
		Id: V1MembershipEntitlementID(resource.Id.Resource),
	})
	member := sdk.NewAssignmentEntitlement(resource, roleMemberEntitlement, resourceTypeUser)
	member.Description = fmt.Sprintf("Has the %s role in Cloudflare", resource.DisplayName)
	member.Annotations = annos
	member.DisplayName = fmt.Sprintf("%s Role Member", resource.DisplayName)
	return []*v2.Entitlement{member}, "", nil, nil
}

func (o *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	page, err := convertPageToken(pt.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	users, resp, err := o.api.AccountMembers(ctx, o.accountId, pageOpts)
	if err != nil {
		return nil, "", nil, err
	}

	roleId := resource.Id.Resource
	nextPage := convertNextPageToken(resp.Page, len(users))
	for _, user := range users {
		userPos := slices.IndexFunc(user.Roles, func(r cloudflare.AccountRole) bool {
			return r.ID == roleId
		})
		if userPos == -1 {
			continue
		}

		roleName := user.Roles[userPos].Name
		v1Identifier := &v2.V1Identifier{
			Id: V1GrantID(V1MembershipEntitlementID(roleId), user.ID),
		}
		uID, err := sdk.NewResourceID(resourceTypeUser, user.User.ID)
		if err != nil {
			return nil, "", nil, err
		}
		grant := sdk.NewGrant(resource, roleName, uID)
		annos := annotations.Annotations(grant.Annotations)
		annos.Update(v1Identifier)
		grant.Annotations = annos
		rv = append(rv, grant)
	}

	return rv, nextPage, nil, nil
}

func roleBuilder(api *cloudflare.API, accountId string) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		api:          api,
		accountId:    accountId,
	}
}

func roleProfile(ctx context.Context, role cloudflare.AccountRole) map[string]interface{} {
	profile := make(map[string]interface{})
	profile["role_id"] = role.ID
	profile["role_name"] = role.Name
	profile["role_description"] = role.Description
	return profile
}
