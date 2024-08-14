package main

import (
	"github.com/conductorone/baton-cloudflare/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	apiKeyField         = field.StringField(connector.ApiKey, field.WithDescription("The api key for the Cloudflare account."))
	apiTokenField       = field.StringField(connector.ApiToken, field.WithDescription("The api token for the Cloudflare account."))
	accountIdField      = field.StringField(connector.AccountId, field.WithRequired(true), field.WithDescription("The account id for the Cloudflare account."))
	emailIdField        = field.StringField(connector.EmailId, field.WithRequired(true), field.WithDescription("The email id for the Cloudflare account."))
	configurationFields = []field.SchemaField{apiKeyField, apiTokenField, accountIdField, emailIdField}
	fieldRelationships  = []field.SchemaFieldRelationship{
		field.FieldsAtLeastOneUsed(apiTokenField, apiKeyField),
		field.FieldsDependentOn(
			[]field.SchemaField{apiKeyField},
			[]field.SchemaField{emailIdField},
		),
	}
)
