package connector

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const (
	orgMemberEntitlement = "member"
	orgMembersPerPage    = 50
)

type organizationResourceType struct {
	resourceType   *v2.ResourceType
	client         *cloudflare.API
	httpClient     *uhttp.BaseHttpClient
	organizationId string
	emailId        string
}

func (o *organizationResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

type cfOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type cfOrganizationResponse struct {
	Result  cfOrganization           `json:"result"`
	Success bool                     `json:"success"`
	Errors  []cloudflare.ResponseInfo `json:"errors"`
}

type cfOrgMemberUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type cfOrgMember struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	User   cfOrgMemberUser `json:"user"`
}

type cfOrgMembersResponse struct {
	Result     []cfOrgMember             `json:"result"`
	ResultInfo cloudflare.ResultInfo     `json:"result_info"`
	Success    bool                      `json:"success"`
	Errors     []cloudflare.ResponseInfo `json:"errors"`
}

func (o *organizationResourceType) ensureHTTPClient(ctx context.Context) error {
	if o.httpClient != nil {
		return nil
	}
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return fmt.Errorf("baton-cloudflare: failed to create http client: %w", err)
	}
	o.httpClient = uhttp.NewBaseHttpClient(httpClient)
	return nil
}

func (o *organizationResourceType) authOpts() []uhttp.RequestOption {
	var opts []uhttp.RequestOption
	if o.client.APIToken != "" {
		opts = append(opts, uhttp.WithBearerToken(o.client.APIToken))
	}
	if o.emailId != "" {
		opts = append(opts, uhttp.WithHeader(XAuthEmailHeaderKey, o.emailId))
	}
	if o.client.APIKey != "" {
		opts = append(opts, uhttp.WithHeader(XAuthKeyHeaderKey, o.client.APIKey))
	}
	return opts
}

func (o *organizationResourceType) getOrganization(ctx context.Context) (*cfOrganization, error) {
	if err := o.ensureHTTPClient(ctx); err != nil {
		return nil, err
	}

	endpointURL, err := url.JoinPath(o.client.BaseURL, "organizations", o.organizationId)
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to build organization endpoint url: %w", err)
	}
	uri, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to parse organization endpoint url: %w", err)
	}

	reqOpts := append([]uhttp.RequestOption{uhttp.WithAcceptJSONHeader()}, o.authOpts()...)
	req, err := o.httpClient.NewRequest(ctx, http.MethodGet, uri, reqOpts...)
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to create organization request: %w", err)
	}

	var result cfOrganizationResponse
	resp, err := o.httpClient.Do(req, uhttp.WithJSONResponse(&result))
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to get organization: %w", err)
	}
	defer resp.Body.Close()

	if !result.Success {
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("baton-cloudflare: get organization failed: %s (code %d)", result.Errors[0].Message, result.Errors[0].Code)
		}
		return nil, fmt.Errorf("baton-cloudflare: get organization failed: unknown error")
	}

	return &result.Result, nil
}

func (o *organizationResourceType) listOrganizationMembers(ctx context.Context, page, perPage int) (*cfOrgMembersResponse, error) {
	if err := o.ensureHTTPClient(ctx); err != nil {
		return nil, err
	}

	endpointURL, err := url.JoinPath(o.client.BaseURL, "organizations", o.organizationId, "members")
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to build org members endpoint url: %w", err)
	}
	uri, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to parse org members endpoint url: %w", err)
	}
	q := uri.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	uri.RawQuery = q.Encode()

	reqOpts := append([]uhttp.RequestOption{uhttp.WithAcceptJSONHeader()}, o.authOpts()...)
	req, err := o.httpClient.NewRequest(ctx, http.MethodGet, uri, reqOpts...)
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to create org members request: %w", err)
	}

	var result cfOrgMembersResponse
	resp, err := o.httpClient.Do(req, uhttp.WithJSONResponse(&result))
	if err != nil {
		return nil, fmt.Errorf("baton-cloudflare: failed to list organization members: %w", err)
	}
	defer resp.Body.Close()

	if !result.Success {
		if len(result.Errors) > 0 {
			return nil, fmt.Errorf("baton-cloudflare: list organization members failed: %s (code %d)", result.Errors[0].Message, result.Errors[0].Code)
		}
		return nil, fmt.Errorf("baton-cloudflare: list organization members failed: unknown error")
	}

	return &result, nil
}

func (o *organizationResourceType) List(ctx context.Context, _ *v2.ResourceId, _ rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	org, err := o.getOrganization(ctx)
	if err != nil {
		return nil, nil, err
	}

	profile := map[string]interface{}{
		"organization_id":   org.ID,
		"organization_name": org.Name,
	}

	resource, err := rs.NewGroupResource(
		org.Name,
		resourceTypeOrganization,
		org.ID,
		[]rs.GroupTraitOption{rs.WithGroupProfile(profile)},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-cloudflare: failed to create organization resource: %w", err)
	}

	return []*v2.Resource{resource}, &rs.SyncOpResults{}, nil
}

func (o *organizationResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	rv := []*v2.Entitlement{
		ent.NewAssignmentEntitlement(
			resource,
			orgMemberEntitlement,
			ent.WithGrantableTo(resourceTypeUser),
			ent.WithDisplayName(fmt.Sprintf("%s Organization Member", resource.DisplayName)),
			ent.WithDescription(fmt.Sprintf("Member of the %s Cloudflare organization", resource.DisplayName)),
		),
	}

	return rv, &rs.SyncOpResults{}, nil
}

func (o *organizationResourceType) Grants(ctx context.Context, resource *v2.Resource, opts rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	page, err := convertPageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-cloudflare: invalid page token error")
	}

	resp, err := o.listOrganizationMembers(ctx, page, orgMembersPerPage)
	if err != nil {
		return nil, nil, err
	}

	var rv []*v2.Grant
	for _, member := range resp.Result {
		if member.Status != "active" {
			continue
		}
		if member.User.ID == "" {
			continue
		}

		userResourceId := &v2.ResourceId{
			ResourceType: resourceTypeUser.Id,
			Resource:     member.User.ID,
		}

		rv = append(rv, grant.NewGrant(resource, orgMemberEntitlement, userResourceId))
	}

	nextPage := convertNextPageToken(resp.ResultInfo.Page, len(resp.Result))

	return rv, &rs.SyncOpResults{NextPageToken: nextPage}, nil
}

func organizationBuilder(client *cloudflare.API, organizationId, emailId string) *organizationResourceType {
	return &organizationResourceType{
		resourceType:   resourceTypeOrganization,
		client:         client,
		organizationId: organizationId,
		emailId:        emailId,
	}
}
