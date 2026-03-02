package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	apiKeyField = field.StringField(
		"api-key",
		field.WithDisplayName("API Key"),
		field.WithDescription("The api key for the Cloudflare account."),
		field.WithIsSecret(true),
		field.WithRequired(true),
	)
	apiTokenField = field.StringField(
		"api-token",
		field.WithDisplayName("API Token"),
		field.WithDescription("The api token for the Cloudflare account."),
		field.WithIsSecret(true),
		field.WithRequired(true),
	)
	accountIdField = field.StringField(
		"account-id",
		field.WithDisplayName("Account ID"),
		field.WithDescription("The account id for the Cloudflare account."),
		field.WithRequired(true),
	)
	emailIdField = field.StringField(
		"email-id",
		field.WithDisplayName("Email ID"),
		field.WithDescription("The email id for the Cloudflare account."),
		field.WithRequired(true),
	)
	configurationFields = []field.SchemaField{
		apiKeyField,
		apiTokenField,
		accountIdField,
		emailIdField,
	}
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	configurationFields,
	field.WithConnectorDisplayName("Cloudflare"),
	field.WithHelpUrl("/docs/baton/cloudflare"),
	field.WithIconUrl("/static/app-icons/cloudflare.svg"),
	field.WithFieldGroups([]field.SchemaFieldGroup{
		{
			Name:        "api-token-group",
			DisplayName: "API Token",
			HelpText:    "Use an API token for authentication.",
			Fields:      []field.SchemaField{accountIdField, apiTokenField},
		},
		{
			Name:        "api-key-group",
			DisplayName: "Email + API key",
			HelpText:    "Use an API key with email for authentication.",
			Fields:      []field.SchemaField{accountIdField, emailIdField, apiKeyField},
		},
	}),
)
