package connector

import (
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPITokenResource(t *testing.T) {
	issued := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	expires := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	token := cloudflare.APIToken{
		ID:        "0123456789abcdef0123456789abcdef",
		Name:      "ci-deploy-token",
		Status:    "active",
		IssuedOn:  &issued,
		ExpiresOn: &expires,
	}

	resource, err := apiTokenResource(token)
	require.NoError(t, err)
	assert.Equal(t, token.ID, resource.GetId().GetResource())
	assert.Equal(t, resourceTypeAPIToken.GetId(), resource.GetId().GetResourceType())
	assert.Equal(t, token.Name, resource.GetDisplayName())

	secretTrait := &v2.SecretTrait{}
	annos := annotations.Annotations(resource.GetAnnotations())
	ok, err := annos.Pick(secretTrait)
	require.NoError(t, err)
	require.True(t, ok, "expected a SecretTrait on the api_token resource")

	assert.Equal(t, v2.SecretTrait_CREDENTIAL_TYPE_STATIC_SECRET, secretTrait.GetCredentialType())
	assert.Equal(t, apiTokenSecretDetail, secretTrait.GetCredentialDetail())
	assert.Equal(t, issued, secretTrait.GetCreatedAt().AsTime())
	assert.Equal(t, expires, secretTrait.GetExpiresAt().AsTime())
}

func TestAPITokenResourceFallbackDisplayName(t *testing.T) {
	token := cloudflare.APIToken{ID: "abc123", Status: "active"}

	resource, err := apiTokenResource(token)
	require.NoError(t, err)
	assert.Equal(t, token.ID, resource.GetDisplayName())
}
