package connector

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

const (
	roleMemberEntitlement = "member"
	// The list custom roles endpoint does not return the super admin role, so we are manually adding it with Cloudflares super admin role ID.
	SuperAdminRoleId    = "33666b9c79b9a5273fc7344ff42f953d"
	errMissingAccountID = "required missing account ID"
	XAuthEmailHeaderKey = "X-Auth-Email"
	XAuthKeyHeaderKey   = "X-Auth-Key"
	NF                  = -1
)

var ErrMissingAccountID = errors.New(errMissingAccountID)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *cloudflare.API
	httpClient   *uhttp.BaseHttpClient
	accountId    string
	emailId      string
}

func (o *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// getRoleResource creates a new connector resource for a Zendesk role.
func roleResource(role cloudflare.AccountRole, resourceTypeRole *v2.ResourceType, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"role_id":   role.ID,
		"role_name": role.Name,
	}

	roleTraitOptions := []rs.RoleTraitOption{
		rs.WithRoleProfile(profile),
	}

	ret, err := rs.NewRoleResource(
		role.Name,
		resourceTypeRole,
		role.ID,
		roleTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *roleResourceType) List(ctx context.Context, _ *v2.ResourceId, pt *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	roles, err := o.client.AccountRoles(ctx, o.accountId)
	if err != nil {
		return nil, "", nil, err
	}
	rv := make([]*v2.Resource, 0, len(roles))
	for _, role := range roles {
		roleResource, err := roleResource(role, resourceTypeRole, nil)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, roleResource)
	}

	// adminRoleResource, err := sdk.NewRoleResource("Super Administrator - All Privileges", resourceTypeRole, nil, SuperAdminRoleId, nil, annos)
	adminRoleResource, err := roleResource(cloudflare.AccountRole{
		ID:   SuperAdminRoleId,
		Name: "Super Administrator - All Privileges",
	}, resourceTypeRole, nil)
	if err != nil {
		return nil, "", nil, err
	}
	rv = append(rv, adminRoleResource)

	return rv, "", nil, nil
}

func (r *roleResourceType) Entitlements(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var (
		rv      []*v2.Entitlement
		options []ent.EntitlementOption
	)
	roles, err := r.client.AccountRoles(ctx, r.accountId)
	if err != nil {
		return nil, "", nil, wrapError(err, "failed to list roles")
	}

	for _, role := range roles {
		options = []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeRole),
			ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, role.Name)),
			ent.WithDescription(fmt.Sprintf("%s of %s Cloudflare role", role.Name, resource.DisplayName)),
		}
		rv = append(rv, ent.NewAssignmentEntitlement(resource, role.Name, options...))
	}

	options = []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeRole),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, roleMemberEntitlement)),
		ent.WithDescription(fmt.Sprintf("%s of %s Cloudflare role", roleMemberEntitlement, resource.DisplayName)),
	}
	rv = append(rv, ent.NewAssignmentEntitlement(resource, roleMemberEntitlement, options...))

	return rv, "", nil, nil
}

// GetAccountMember returns an account member.
func (r *roleResourceType) GetAccountMember(ctx context.Context, accountID string, memberID string) (*cloudflare.AccountMemberDetailResponse, error) {
	var accountMemberListResponse = &cloudflare.AccountMemberDetailResponse{}
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	if accountID == "" {
		return &cloudflare.AccountMemberDetailResponse{}, ErrMissingAccountID
	}

	r.httpClient = uhttp.NewBaseHttpClient(httpClient)
	endpointUrl := fmt.Sprintf("%s/accounts/%s/members/%s", r.client.BaseURL, accountID, memberID)
	uri, err := url.Parse(endpointUrl)
	if err != nil {
		return nil, err
	}

	req, err := r.httpClient.NewRequest(ctx,
		http.MethodGet,
		uri,
		uhttp.WithAcceptJSONHeader(),
		WithAuthorizationBearerHeader(r.client.APIToken),
		uhttp.WithHeader(XAuthEmailHeaderKey, r.emailId),
		uhttp.WithHeader(XAuthKeyHeaderKey, r.client.APIKey),
	)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req, uhttp.WithJSONResponse(&accountMemberListResponse))
	if err != nil {
		return nil, fmt.Errorf("%s %s", err.Error(), resp.Body)
	}

	defer resp.Body.Close()
	return accountMemberListResponse, err
}

func (r *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	page, err := convertPageToken(pt.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Cloudflare: invalid page token error")
	}

	pageOpts := cloudflare.PaginationOptions{Page: page}
	users, resp, err := r.client.AccountMembers(ctx, r.accountId, pageOpts)
	if err != nil {
		return nil, "", nil, err
	}

	roleId := resource.Id.Resource
	nextPage := convertNextPageToken(resp.Page, len(users))
	for _, user := range users {
		userPos := slices.IndexFunc(user.Roles, func(r cloudflare.AccountRole) bool {
			return r.ID == roleId
		})
		if userPos == NF {
			continue
		}

		roleName := user.Roles[userPos].Name
		accUser := cloudflare.AccountMember{
			User: cloudflare.AccountMemberUserDetails{
				ID:        user.User.ID,
				FirstName: user.User.FirstName,
				LastName:  user.User.LastName,
				Email:     user.User.Email,
			},
		}
		ur, err := userResource(accUser)
		if err != nil {
			return nil, "", nil, wrapError(err, "failed to create user resource")
		}

		gr := grant.NewGrant(resource, roleName, ur.Id)
		annos := annotations.Annotations(gr.Annotations)
		v1Identifier := &v2.V1Identifier{
			Id: V1GrantID(V1MembershipEntitlementID(roleId), user.ID),
		}
		annos.Update(v1Identifier)
		gr.Annotations = annos
		rv = append(rv, gr)
	}

	return rv, nextPage, nil, nil
}

func (r *roleResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	var (
		err    error
		userId = principal.Id.Resource
		roleId = entitlement.Resource.Id.Resource
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
		return nil, fmt.Errorf("error: %s", err.Error())
	}

	roles := []cloudflare.AccountRole{{
		ID: roleId},
	}
	for _, role := range account.Result.Roles {
		if role.ID == roleId {
			l.Warn(
				"cloudflare-connector: user already has this role",
				zap.String("principal_id", principal.Id.String()),
				zap.String("principal_type", principal.Id.ResourceType),
			)
			return nil, fmt.Errorf("cloudflare-connector: user %s already has this role", principal.DisplayName)
		}

		roles = append(roles, cloudflare.AccountRole{
			ID: role.ID,
		})
	}

	member, err := r.UpdateAccountMember(ctx, r.accountId, memberId, cloudflare.AccountMember{
		Roles: roles,
	})
	err = getError(err)
	if err != nil {
		return nil, err
	}

	l.Warn("Role has been created.",
		zap.String("ID", member.ID),
		zap.String("Status", member.Status),
	)

	return nil, nil
}

// UpdateAccountMember
// Modify an account member
// https://developers.cloudflare.com/api/operations/account-members-update-member
func (r *roleResourceType) UpdateAccountMember(ctx context.Context, accountID, memberID string, accountMemberRoles cloudflare.AccountMember) (*cloudflare.AccountMember, error) {
	var (
		accountMemberListResponse = &Response{}
		body                      struct {
			Roles []roles
		}
	)
	for _, role := range accountMemberRoles.Roles {
		body.Roles = append(body.Roles, roles{
			ID: role.ID,
		})
	}

	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	if accountID == "" {
		return nil, ErrMissingAccountID
	}

	r.httpClient = uhttp.NewBaseHttpClient(httpClient)
	endpointUrl := fmt.Sprintf("%s/accounts/%s/members/%s", r.client.BaseURL, accountID, memberID)
	uri, err := url.Parse(endpointUrl)
	if err != nil {
		return nil, err
	}

	req, err := r.httpClient.NewRequest(ctx,
		http.MethodPut,
		uri,
		uhttp.WithJSONBody(body),
		uhttp.WithAcceptJSONHeader(),
		WithAuthorizationBearerHeader(r.client.APIToken),
		uhttp.WithHeader(XAuthEmailHeaderKey, r.emailId),
		uhttp.WithHeader(XAuthKeyHeaderKey, r.client.APIKey),
	)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req, uhttp.WithJSONResponse(&accountMemberListResponse))
	if err != nil {
		ce := &CloudflareError{
			ErrorMessage:     err.Error(),
			ErrorDescription: err.Error(),
			ErrorLink:        endpointUrl,
		}
		if resp != nil {
			ce.ErrorCode = resp.StatusCode
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				ce.ErrorSummary = fmt.Sprintf("Error reading response body %s", err.Error())
				return nil, ce
			}

			ce.ErrorSummary = string(bodyBytes)
		}

		return nil, ce
	}

	defer resp.Body.Close()
	return &accountMemberListResponse.Result, nil
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

	index := slices.IndexFunc(account.Result.Roles, func(c cloudflare.AccountRole) bool {
		return c.ID == roleId
	})
	if index == NF {
		l.Warn(
			"cloudflare-connector: user does not have this role",
			zap.String("principal_id", principal.Id.String()),
			zap.String("principal_type", principal.Id.ResourceType),
		)
		return nil, fmt.Errorf("cloudflare-connector: user %s does not have this role", principal.DisplayName)
	}

	member, err := r.UpdateAccountMember(ctx, r.accountId, memberId, cloudflare.AccountMember{
		Roles: roles,
	})
	err = getError(err)
	if err != nil {
		return nil, err
	}

	l.Warn("Role has been revoked.",
		zap.String("ID", member.ID),
		zap.String("Status", member.Status),
	)

	return nil, nil
}

func roleBuilder(client *cloudflare.API, accountId, emailId string) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
		accountId:    accountId,
		emailId:      emailId,
	}
}
