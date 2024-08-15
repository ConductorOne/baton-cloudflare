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

var (
	apiKeyField         = field.StringField(apiKey, field.WithDescription("The api key for the Cloudflare account."))
	apiTokenField       = field.StringField(apiToken, field.WithDescription("The api token for the Cloudflare account."))
	accountIdField      = field.StringField(accountId, field.WithRequired(true), field.WithDescription("The account id for the Cloudflare account."))
	emailIdField        = field.StringField(emailId, field.WithDescription("The email id for the Cloudflare account."))
	configurationFields = []field.SchemaField{apiKeyField, apiTokenField, accountIdField, emailIdField}
	fieldRelationships  = []field.SchemaFieldRelationship{
		field.FieldsAtLeastOneUsed(apiTokenField, apiKeyField),
		field.FieldsDependentOn(
			[]field.SchemaField{apiKeyField},
			[]field.SchemaField{emailIdField},
		),
	}
)

const (
	version       = "dev"
	connectorName = "baton-cloudflare"
	apiKey        = "api-key"
	apiToken      = "api-token"
	accountId     = "account-id"
	emailId       = "email-id"
)

func main() {
	ctx := context.Background()
	_, cmd, err := configSchema.DefineConfiguration(ctx,
		connectorName,
		getConnector,
		field.NewConfiguration(configurationFields, fieldRelationships...),
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
	config := connector.Config{
		AccountId: cfg.GetString(accountId),
		ApiToken:  cfg.GetString(apiToken),
		EmailId:   cfg.GetString(emailId),
		ApiKey:    cfg.GetString(apiKey),
	}

	cb, err := connector.New(ctx, config)
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
