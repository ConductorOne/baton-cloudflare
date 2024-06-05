package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/sdk"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	roleMemberEntitlement = "member"
	// The list custom roles endpoint does not return the super admin role, so we are manually adding it with Cloudflares super admin role ID.
	SuperAdminRoleId    = "33666b9c79b9a5273fc7344ff42f953d"
	errMissingAccountID = "required missing account ID"
)

var ErrMissingAccountID = errors.New(errMissingAccountID)

type roleResourceType struct {
	resourceType *v2.ResourceType
	// api          *cloudflare.API
	client     *cloudflare.API
	httpClient *http.Client
	accountId  string
}

func (o *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func (o *roleResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	roles, err := o.client.AccountRoles(ctx, o.accountId)
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

// GetAccountMember returns an account member.
func (r *roleResourceType) GetAccountMember(ctx context.Context, accountID string, memberID string) (*cloudflare.AccountMemberDetailResponse, error) {
	var accountMemberListResponse = &cloudflare.AccountMemberDetailResponse{}
	if accountID == "" {
		return &cloudflare.AccountMemberDetailResponse{}, ErrMissingAccountID
	}
	r.httpClient = &http.Client{}
	requestURL := fmt.Sprintf("%s/accounts/%s/members/%s", r.client.BaseURL, accountID, memberID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return &cloudflare.AccountMemberDetailResponse{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-Auth-Email", r.client.APIEmail)
	req.Header.Add("X-Auth-Key", r.client.APIKey)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return &cloudflare.AccountMemberDetailResponse{}, err
	}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(accountMemberListResponse)
	if err != nil {
		return &cloudflare.AccountMemberDetailResponse{}, err
	}

	return accountMemberListResponse, err
}

func (o *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	page, err := convertPageToken(pt.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	users, resp, err := o.client.AccountMembers(ctx, o.accountId, pageOpts)
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

func (r *roleResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	var (
		err    error
		userId = principal.Id.Resource
	)
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"baton-cloudflare: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-cloudflare: only users can be granted role membership")
	}

	memberId, err := getMemberId(ctx, r, userId)
	if err != nil {
		return nil, err
	}

	account, err := r.GetAccountMember(ctx, r.accountId, memberId)
	if err != nil {
		return nil, err
	}

	roles := []cloudflare.AccountRole{{
		ID: entitlement.Resource.Id.Resource},
	}
	for _, role := range account.Result.Roles {
		roles = append(roles, cloudflare.AccountRole{
			ID: role.ID,
		})
	}

	member, err := r.client.UpdateAccountMember(ctx, r.accountId, memberId, cloudflare.AccountMember{
		Roles: roles,
	})
	if err != nil {
		return nil, err
	}

	l.Warn("Role has been created.",
		zap.String("ID", member.ID),
		zap.String("Status", member.Status),
	)

	return nil, nil
}

func getMemberId(ctx context.Context, r *roleResourceType, userId string) (string, error) {
	memberUsers, _, err := r.client.AccountMembers(ctx, r.accountId, cloudflare.PaginationOptions{})
	if err != nil {
		return "", wrapError(err, "failed to list user members")
	}

	for _, memberUser := range memberUsers {
		if memberUser.User.ID == userId {
			return memberUser.ID, nil
		}
	}

	return "", nil
}

func (r *roleResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	entitlement := grant.Entitlement
	principal := grant.Principal

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"couldflare-connector: only users can have role membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("couldflare-connector: only users can have role membership revoked")
	}

	userId := principal.Id.Resource
	roleId := entitlement.Resource.Id.Resource

	memberId, err := getMemberId(ctx, r, userId)
	if err != nil {
		return nil, err
	}

	account, err := r.GetAccountMember(ctx, r.accountId, memberId)
	if err != nil {
		return nil, err
	}

	roles := []cloudflare.AccountRole{}
	for _, role := range account.Result.Roles {
		if roleId != role.ID {
			roles = append(roles, cloudflare.AccountRole{
				ID: role.ID,
			})
		}
	}

	member, err := r.client.UpdateAccountMember(ctx, r.accountId, memberId, cloudflare.AccountMember{
		Roles: roles,
	})
	if err != nil {
		return nil, err
	}

	l.Warn("Role has been revoked.",
		zap.String("ID", member.ID),
		zap.String("Status", member.Status),
	)

	return nil, nil
}

func roleBuilder(api *cloudflare.API, accountId string) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       api,
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
