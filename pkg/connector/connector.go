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

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: v1AnnotationsForResourceType("user"),
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
		Annotations: v1AnnotationsForResourceType("role"),
	}
)

type Config struct {
	AccountId string
	ApiKey    string
}

type Cloudflare struct {
	api       *cloudflare.API
	accountId string
}

func New(ctx context.Context, config Config) (*Cloudflare, error) {
	var api *cloudflare.API
	if config.AccountId != "" && config.ApiKey != "" {
		httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, nil))
		if err != nil {
			return nil, err
		}

		api, err = cloudflare.NewWithAPIToken(config.ApiKey, cloudflare.HTTPClient(httpClient))
		if err != nil {
			return nil, err
		}
	}

	return &Cloudflare{
		api:       api,
		accountId: config.AccountId,
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
		_, _, err := c.api.Account(ctx, c.accountId)
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
	rs = append(rs, userBuilder(c.api, c.accountId), roleBuilder(c.api, c.accountId))
	return rs
}
