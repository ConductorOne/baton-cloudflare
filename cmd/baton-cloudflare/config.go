package main

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	apiKeyField = field.StringField(
		"api-key",
		field.WithDescription("The api key for the Cloudflare account."),
	)
	apiTokenField = field.StringField(
		"api-token",
		field.WithDescription("The api token for the Cloudflare account."),
	)
	accountIdField = field.StringField(
		"account-id",
		field.WithRequired(true),
		field.WithDescription("The account id for the Cloudflare account."),
	)
	emailIdField = field.StringField(
		"email-id",
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
