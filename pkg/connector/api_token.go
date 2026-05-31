package connector

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

const (
	// apiTokenSecretDetail is the §2.8 axis-2 detail string for account-owned API tokens.
	apiTokenSecretDetail = "cloudflare.account_api_token" //nolint:gosec // axis-2 detail label, not a credential value
	apiTokensPerPage     = 50
)

type apiTokenResourceType struct {
	resourceType *v2.ResourceType
	client       *cloudflare.API
	httpClient   *uhttp.BaseHttpClient
	accountId    string
	emailId      string
}

func (o *apiTokenResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// accountAPITokenListResponse models GET /accounts/{account_id}/tokens. Cloudflare
// returns token metadata only; the secret value is never present on list responses.
type accountAPITokenListResponse struct {
	Result     []cloudflare.APIToken     `json:"result"`
	ResultInfo cloudflare.ResultInfo     `json:"result_info"`
	Success    bool                      `json:"success"`
	Errors     []cloudflare.ResponseInfo `json:"errors"`
}

func apiTokenResource(token cloudflare.APIToken) (*v2.Resource, error) {
	secretTraitOpts := []rs.SecretTraitOption{
		rs.WithSecretType(v2.SecretTrait_CREDENTIAL_TYPE_STATIC_SECRET),
		rs.WithSecretDetail(apiTokenSecretDetail),
	}
	if token.IssuedOn != nil {
		secretTraitOpts = append(secretTraitOpts, rs.WithSecretCreatedAt(*token.IssuedOn))
	}
	if token.ExpiresOn != nil {
		secretTraitOpts = append(secretTraitOpts, rs.WithSecretExpiresAt(*token.ExpiresOn))
	}

	displayName := token.Name
	if displayName == "" {
		displayName = token.ID
	}

	return rs.NewSecretResource(displayName, resourceTypeAPIToken, token.ID, secretTraitOpts)
}

func (o *apiTokenResourceType) List(ctx context.Context, _ *v2.ResourceId, opts rs.SyncOpAttrs) ([]*v2.Resource, *rs.SyncOpResults, error) {
	if o.accountId == "" {
		return nil, nil, ErrMissingAccountID
	}

	page, err := convertPageToken(opts.PageToken.Token)
	if err != nil {
		return nil, nil, fmt.Errorf("cloudflare: invalid page token error")
	}

	resp, err := o.listAccountAPITokens(ctx, page, apiTokensPerPage)
	if err != nil {
		return nil, nil, err
	}

	rv := make([]*v2.Resource, 0, len(resp.Result))
	for _, token := range resp.Result {
		tokenResource, err := apiTokenResource(token)
		if err != nil {
			return nil, nil, err
		}
		rv = append(rv, tokenResource)
	}

	nextPage := ""
	if resp.ResultInfo.PerPage > 0 && resp.ResultInfo.Page*resp.ResultInfo.PerPage < resp.ResultInfo.Total {
		nextPage = strconv.Itoa(page + 1)
	}

	return rv, &rs.SyncOpResults{NextPageToken: nextPage}, nil
}

func (o *apiTokenResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Entitlement, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

func (o *apiTokenResourceType) Grants(_ context.Context, _ *v2.Resource, _ rs.SyncOpAttrs) ([]*v2.Grant, *rs.SyncOpResults, error) {
	return nil, nil, nil
}

// listAccountAPITokens calls GET /accounts/{account_id}/tokens. cloudflare-go's
// APITokens helper only covers /user/tokens, so account-owned tokens are fetched
// directly, reusing the same auth headers the rest of the connector relies on.
func (o *apiTokenResourceType) listAccountAPITokens(ctx context.Context, page, perPage int) (*accountAPITokenListResponse, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}
	o.httpClient = uhttp.NewBaseHttpClient(httpClient)

	endpointURL := fmt.Sprintf("%s/accounts/%s/tokens", o.client.BaseURL, o.accountId)
	uri, err := url.Parse(endpointURL)
	if err != nil {
		return nil, err
	}
	q := uri.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	uri.RawQuery = q.Encode()

	reqOpts := []uhttp.RequestOption{
		uhttp.WithAcceptJSONHeader(),
	}
	if o.client.APIToken != "" {
		reqOpts = append(reqOpts, uhttp.WithBearerToken(o.client.APIToken))
	}
	if o.emailId != "" {
		reqOpts = append(reqOpts, uhttp.WithHeader(XAuthEmailHeaderKey, o.emailId))
	}
	if o.client.APIKey != "" {
		reqOpts = append(reqOpts, uhttp.WithHeader(XAuthKeyHeaderKey, o.client.APIKey))
	}

	req, err := o.httpClient.NewRequest(ctx, http.MethodGet, uri, reqOpts...)
	if err != nil {
		return nil, err
	}

	var result accountAPITokenListResponse
	resp, err := o.httpClient.Do(req, uhttp.WithJSONResponse(&result))
	if err != nil {
		return nil, fmt.Errorf("cloudflare: failed to list account API tokens: %w", err)
	}
	defer resp.Body.Close()

	return &result, nil
}

func apiTokenBuilder(client *cloudflare.API, accountId, emailId string) *apiTokenResourceType {
	return &apiTokenResourceType{
		resourceType: resourceTypeAPIToken,
		client:       client,
		accountId:    accountId,
		emailId:      emailId,
	}
}
