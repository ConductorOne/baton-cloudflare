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

func New(ctx context.Context, config Config) (*Cloudflare, error) {
	var (
		client *cloudflare.API
		err    error
	)

	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, nil))
	if err != nil {
		return nil, err
	}

	if config.ApiToken != "" {
		client, err = cloudflare.NewWithAPIToken(config.ApiToken, cloudflare.HTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
	}

	if config.ApiKey != "" && config.EmailId != "" {
		client, err = cloudflare.New(config.ApiKey, config.EmailId, cloudflare.HTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
	}

	return &Cloudflare{
		client:    client,
		accountId: config.AccountId,
		emailId:   config.EmailId,
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
	rs := []connectorbuilder.ResourceSyncer{}
	rs = append(rs, userBuilder(c.client, c.accountId))
	rs = append(rs, roleBuilder(c.client, c.accountId, c.emailId))
	return rs
}
