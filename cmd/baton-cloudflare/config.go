package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/spf13/cobra"
)

// config defines the external configuration required for the connector to run.
type config struct {
	cli.BaseConfig `mapstructure:",squash"` // Puts the base config options in the same place as the connector options

	ApiToken  string `mapstructure:"api-token"`
	ApiKey    string `mapstructure:"api-key"`
	AccountId string `mapstructure:"account-id"`
	EmailId   string `mapstructure:"email-id"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.ApiToken == "" {
		return fmt.Errorf("api key is missing")
	}
	if cfg.AccountId == "" {
		return fmt.Errorf("account id is missing")
	}
	if cfg.EmailId == "" {
		return fmt.Errorf("email id is missing")
	}
	return nil
}

// cmdFlags sets the cmdFlags required for the connector.
func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("api-token", "", "The api torkn for the Cloudflare account. ($BATON_API_TOKEN)")
	cmd.PersistentFlags().String("api-key", "", "The api key for the Cloudflare account. ($BATON_API_KEY)")
	cmd.PersistentFlags().String("account-id", "", "The account id for the Cloudflare account. ($BATON_ACCOUNT_ID)")
	cmd.PersistentFlags().String("email-id", "", "The email id for the Cloudflare account. ($BATON_EMAIL_ID)")
}
