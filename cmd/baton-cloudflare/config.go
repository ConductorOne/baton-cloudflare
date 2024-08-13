package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-cloudflare/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/spf13/viper"
)

var (
	ApiKey              = field.StringField(connector.ApiKey, field.WithDescription("The api key for the Cloudflare account."))
	ApiToken            = field.StringField(connector.ApiToken, field.WithDescription("The api token for the Cloudflare account."))
	AccountId           = field.StringField(connector.AccountId, field.WithRequired(true), field.WithDescription("The account id for the Cloudflare account."))
	EmailId             = field.StringField(connector.EmailId, field.WithRequired(true), field.WithDescription("The email id for the Cloudflare account."))
	configurationFields = []field.SchemaField{ApiKey, ApiToken, AccountId, EmailId}
)

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *viper.Viper) error {
	if cfg.GetString(connector.AccountId) == "" {
		return fmt.Errorf("account id is missing")
	}

	if cfg.GetString(connector.ApiToken) == "" && cfg.GetString(connector.ApiKey) == "" {
		return fmt.Errorf("either api token or api key must be provided")
	}

	if cfg.GetString(connector.ApiKey) != "" && cfg.GetString(connector.EmailId) == "" {
		return fmt.Errorf("email id is missing")
	}

	return nil
}
