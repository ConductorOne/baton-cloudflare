package connector

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudflare/cloudflare-go"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

func New(ctx context.Context, cfg Config) (*Cloudflare, error) {
	var (
		client    *cloudflare.API
		apiKey    = cfg.ApiKey
		apiToken  = cfg.ApiToken
		accountId = cfg.AccountId
		emailId   = cfg.EmailId
		baseURL   = cfg.BaseURL
		err       error
	)

	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, nil))
	if err != nil {
		return nil, err
	}

	// Build options slice
	opts := []cloudflare.Option{cloudflare.HTTPClient(httpClient)}
	if baseURL != "" {
		opts = append(opts, cloudflare.BaseURL(baseURL))
	}

	if apiToken != "" {
		client, err = cloudflare.NewWithAPIToken(apiToken, opts...)
		if err != nil {
			return nil, err
		}
	}

	if apiKey != "" && emailId != "" {
		client, err = cloudflare.New(apiKey, emailId, opts...)
		if err != nil {
			return nil, err
		}
	}

	return &Cloudflare{
		client:    client,
		accountId: accountId,
		emailId:   emailId,
	}, nil
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

func (c *Cloudflare) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(c.client, c.accountId),
		roleBuilder(c.client, c.accountId, c.emailId),
	}
}
