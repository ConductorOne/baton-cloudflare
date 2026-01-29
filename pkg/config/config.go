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
	)
	apiTokenField = field.StringField(
		"api-token",
		field.WithDisplayName("API Token"),
		field.WithDescription("The api token for the Cloudflare account."),
		field.WithIsSecret(true),
	)
	accountIdField = field.StringField(
		"account-id",
		field.WithDisplayName("Account ID"),
		field.WithRequired(true),
		field.WithDescription("The account id for the Cloudflare account."),
	)
	emailIdField = field.StringField(
		"email-id",
		field.WithDisplayName("Email ID"),
		field.WithDescription("The email id for the Cloudflare account."),
	)
	configurationFields = []field.SchemaField{
		apiKeyField,
		apiTokenField,
		accountIdField,
		emailIdField,
	}
	fieldRelationships = []field.SchemaFieldRelationship{
		field.FieldsAtLeastOneUsed(apiTokenField, apiKeyField),
		field.FieldsDependentOn(
			[]field.SchemaField{apiKeyField},
			[]field.SchemaField{emailIdField},
		),
	}
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	configurationFields,
	field.WithConstraints(fieldRelationships...),
	field.WithConnectorDisplayName("Cloudflare"),
	field.WithHelpUrl("/docs/baton/cloudflare"),
	field.WithIconUrl("/static/app-icons/cloudflare.svg"),
)
