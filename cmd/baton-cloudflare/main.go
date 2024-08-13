package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-cloudflare/pkg/connector"
	configSchema "github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	version       = "dev"
	connectorName = "baton-cloudflare"
)

var (
	ApiKey              = field.StringField(connector.ApiKey, field.WithRequired(true), field.WithDescription("The api key for the Cloudflare account."))
	ApiToken            = field.StringField(connector.ApiToken, field.WithRequired(true), field.WithDescription("The api token for the Cloudflare account."))
	AccountId           = field.StringField(connector.AccountId, field.WithRequired(true), field.WithDescription("The account id for the Cloudflare account."))
	EmailId             = field.StringField(connector.EmailId, field.WithRequired(true), field.WithDescription("The email id for the Cloudflare account."))
	configurationFields = []field.SchemaField{ApiKey, ApiToken, AccountId, EmailId}
)

func main() {
	ctx := context.Background()
	_, cmd, err := configSchema.DefineConfiguration(ctx,
		connectorName,
		getConnector,
		field.NewConfiguration(configurationFields),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version
	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, cfg *viper.Viper) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)
	cb, err := connector.New(ctx, cfg)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	connector, err := connectorbuilder.NewConnector(ctx, cb)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return connector, nil
}
