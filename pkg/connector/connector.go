package connector

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudflare/cloudflare-go"
	cfg "github.com/conductorone/baton-cloudflare/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

func New(ctx context.Context, cc *cfg.Cloudflare, opts *cli.ConnectorOpts) (connectorbuilder.ConnectorBuilderV2, []connectorbuilder.Opt, error) {
	var (
		client    *cloudflare.API
		apiKey    = cc.ApiKey
		apiToken  = cc.ApiToken
		accountId = cc.AccountId
		emailId   = cc.EmailId
		err       error
	)

	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, nil))
	if err != nil {
		return nil, nil, err
	}

	if apiToken != "" {
		client, err = cloudflare.NewWithAPIToken(apiToken, cloudflare.HTTPClient(httpClient))
		if err != nil {
			return nil, nil, err
		}
	}

	if apiKey != "" && emailId != "" {
		client, err = cloudflare.New(apiKey, emailId, cloudflare.HTTPClient(httpClient))
		if err != nil {
			return nil, nil, err
		}
	}

	return &Cloudflare{
		client:    client,
		accountId: accountId,
		emailId:   emailId,
	}, nil, nil
}

func (c *Cloudflare) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	_, err := c.Validate(ctx)
	if err != nil {
		return nil, err
	}

	var annos annotations.Annotations
	annos.Update(&v2.ExternalLink{
		Url: c.accountId,
	})

	return &v2.ConnectorMetadata{
		DisplayName: "Cloudflare",
		Annotations: annos,
	}, nil
}

func (c *Cloudflare) Validate(ctx context.Context) (annotations.Annotations, error) {
	if c.accountId != "" {
		if c.client == nil {
			return nil, fmt.Errorf("Cloudflare: client not configured. API key/email or token not provided")
		}

		_, _, err := c.client.Account(ctx, c.accountId)
		if err != nil {
			return nil, fmt.Errorf("Cloudflare: failed to validate API keys: %w", err)
		}
	}

	return nil, nil
}

func (c *Cloudflare) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

func (c *Cloudflare) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncerV2 {
	return []connectorbuilder.ResourceSyncerV2{
		userBuilder(c.client, c.accountId),
		roleBuilder(c.client, c.accountId, c.emailId),
	}
}
