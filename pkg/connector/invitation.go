package connector

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type InvitationResourceType struct {
	resourceType *v2.ResourceType
	client       *cloudflare.API
	accountId    string
	emailId      string
}

func (o *InvitationResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func invitationResource(member cloudflare.AccountMember) (*v2.Resource, error) {
	email := member.User.Email
	status := cases.Title(language.English).String(member.Status)
	profile := map[string]interface{}{
		"email":  email,
		"status": status,
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithDetailedStatus(v2.UserTrait_Status_STATUS_ENABLED, status),
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

// listPendingMembers fetches only pending account members from the Cloudflare API using a
// raw HTTP call with ?status=pending.
//
// The cloudflare-go SDK's AccountMembers() only accepts PaginationOptions (page + per_page)
// and does not expose the status query parameter supported by the REST API
// (GET /accounts/{id}/members?status=pending|accepted|rejected).
// Until the SDK adds a dedicated filter params struct we call the endpoint directly so that
// only pending invitations are returned, avoiding a full member scan on every sync.
func (o *InvitationResourceType) listPendingMembers(ctx context.Context, page int) ([]cloudflare.AccountMember, cloudflare.ResultInfo, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, cloudflare.ResultInfo{}, err
	}

	baseClient := uhttp.NewBaseHttpClient(httpClient)

	params := url.Values{}
	params.Set("status", "pending")
	params.Set("page", strconv.Itoa(page))
	endpointURL := fmt.Sprintf("%s/accounts/%s/members?%s", o.client.BaseURL, o.accountId, params.Encode())

	uri, err := url.Parse(endpointURL)
	if err != nil {
		return nil, cloudflare.ResultInfo{}, err
	}

	opts := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithBearerToken(o.client.APIToken),
	}
	if o.emailId != "" {
		opts = append(opts, uhttp.WithHeader(XAuthEmailHeaderKey, o.emailId))
	}
	if o.client.APIKey != "" {
		opts = append(opts, uhttp.WithHeader(XAuthKeyHeaderKey, o.client.APIKey))
	}

	req, err := baseClient.NewRequest(ctx, http.MethodGet, uri, opts...)
	if err != nil {
		return nil, cloudflare.ResultInfo{}, err
	}

	var response cloudflare.AccountMembersListResponse
	resp, err := baseClient.Do(req, uhttp.WithJSONResponse(&response))
	if err != nil {
		return nil, cloudflare.ResultInfo{}, fmt.Errorf("baton-cloudflare: failed to list pending invitations: %w", err)
	}

	defer resp.Body.Close()
	return response.Result, response.ResultInfo, nil
}

func (o *InvitationResourceType) List(ctx context.Context, _ *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	page, err := convertPageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-cloudflare: invalid page token error")
	}

	members, resultInfo, err := o.listPendingMembers(ctx, page)
	if err != nil {
		return nil, nil, err
	}

	nextPage := convertNextPageToken(resultInfo.Page, len(members))
	rv := make([]*v2.Resource, 0, len(members))
	for _, member := range members {
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

func invitationBuilder(client *cloudflare.API, accountId, emailId string) *InvitationResourceType {
	return &InvitationResourceType{
		resourceType: resourceTypeInvitation,
		client:       client,
		accountId:    accountId,
		emailId:      emailId,
	}
}
