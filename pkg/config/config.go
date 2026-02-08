package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	ApiKeyField = field.StringField(
		"api-key",
		field.WithIsSecret(true),
		field.WithDescription("The api key for the Cloudflare account."),
	)
	ApiTokenField = field.StringField(
		"api-token",
		field.WithIsSecret(true),
		field.WithDescription("The api token for the Cloudflare account."),
	)
	AccountIdField = field.StringField(
		"account-id",
		field.WithRequired(true),
		field.WithDescription("The account id for the Cloudflare account."),
	)
	EmailIdField = field.StringField(
		"email-id",
		field.WithDescription("The email id for the Cloudflare account."),
	)

	BaseURLField = field.StringField(
		"base-url",
		field.WithDescription("Override the Cloudflare API URL (for testing)"),
	)

	FieldRelationships = []field.SchemaFieldRelationship{
		field.FieldsAtLeastOneUsed(ApiTokenField, ApiKeyField),
		field.FieldsDependentOn(
			[]field.SchemaField{ApiKeyField},
			[]field.SchemaField{EmailIdField},
		),
	}
)

//go:generate go run ./gen
var Config = field.NewConfiguration([]field.SchemaField{
	ApiKeyField,
	ApiTokenField,
	AccountIdField,
	EmailIdField,
	BaseURLField,
}, field.WithConstraints(FieldRelationships...))

func ValidateConfig(cfg *Cloudflare) error {
	return nil
}
