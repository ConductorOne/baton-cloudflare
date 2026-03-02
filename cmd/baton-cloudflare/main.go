package main

import (
	"context"

	cfg "github.com/conductorone/baton-cloudflare/pkg/config"
	"github.com/conductorone/baton-cloudflare/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
)

var version = "dev"

func main() {
	ctx := context.Background()
	config.RunConnector(ctx,
		"baton-cloudflare",
		version,
		cfg.Config,
		connector.New,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilderV2(&connector.Cloudflare{}),
	)
}
