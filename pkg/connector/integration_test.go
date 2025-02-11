package connector

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/stretchr/testify/assert"
)

var (
	ctx        = context.Background()
	accountID  = "b37e72c7341f3de17a1bfde947cb8f93"
	emailId    = "miguel.angel.chavez.martinez@gmail.com"
	userId     = "9d9a62a5b834a8c9c5cf43cd234dfd4a"
	memberID   = "c03c9ac2229f0bf5d75ef307c10b3b17"
	httpClient = getHttpClientForTesting()
	apiKey     = os.Getenv("BATON_API_KEY")
	apiToken   = os.Getenv("BATON_API_TOKEN")
)

func TestUpdateAccountMember(t *testing.T) {
	var (
		roles   []cloudflare.AccountRole
		baseURL = "https://api.cloudflare.com/client/v4"
		rolesId = []string{
			"35956457e745b2af7331713a1ddf4fdb",
			"08abaa5235c2196d5f3daf457190161b",
			"3a170f9cfd62f321d6d835dc44bfe6dc",
			"6ddc5f80969d01105b5a0931e0079365",
		}
	)
	if apiKey == "" && apiToken == "" {
		t.Skip()
	}
	roleBuilder := getRoleBuilderForTesting(&cloudflare.API{
		APIKey:   apiKey,
		APIToken: apiToken,
		APIEmail: emailId,
		BaseURL:  baseURL,
	})
	for _, role := range rolesId {
		roles = append(roles, cloudflare.AccountRole{
			ID: role,
		})
	}

	accountMember, err := roleBuilder.UpdateAccountMember(ctx, accountID, memberID, cloudflare.AccountMember{
		Roles: roles,
	})
	assert.Nil(t, err)
	assert.NotNil(t, accountMember)
}

func TestResourceTypeGrantFails(t *testing.T) {
	var (
		resourceDisplayName = "API Gateway Read Role API Gateway Read"
		roleEntitlement     = "API Gateway Read"
		userEmail           = "miguel_chavez_m@hotmail.com"
		roleId              = "35956457e745b2af7331713a1ddf4fdb"
		client              *cloudflare.API
	)
	if apiKey == "" && apiToken == "" {
		t.Skip()
	}
	accUser := getAccountMemberForTesting(accountID, userId, userEmail)
	principal, err := userResource(*accUser)
	assert.Nil(t, err)
	role := getRoleForTesting(roleId, resourceDisplayName, roleEntitlement)
	resource, err := roleResource(*role, resourceTypeRole, nil)
	assert.Nil(t, err)
	entitlement := getEntitlementForTesting(resource, resourceDisplayName, roleEntitlement)
	client = getClientForTesting(apiToken, apiKey)
	roleBuilder := getRoleBuilderForTesting(client)
	_, err = roleBuilder.Grant(ctx, principal, entitlement)
	assert.NotNil(t, err)
	errMsg := fmt.Sprintf("cloudflare-connector: user %s already has this role", principal.DisplayName)
	assert.Equal(t, err.Error(), errMsg, errMsg)
}

func TestResourceTypeGrant(t *testing.T) {
	var (
		resourceDisplayName = "Billing Cloudflare role"
		roleEntitlement     = "Billing"
		userEmail           = "miguel_chavez_m@hotmail.com"
		roleId              = "298ce8e7a2ba08b9d18ce0a32bb458ee"
		client              *cloudflare.API
	)
	if apiKey == "" && apiToken == "" {
		t.Skip()
	}
	accUser := getAccountMemberForTesting(accountID, userId, userEmail)
	principal, err := userResource(*accUser)
	assert.Nil(t, err)
	role := getRoleForTesting(roleId, resourceDisplayName, roleEntitlement)
	resource, err := roleResource(*role, resourceTypeRole, nil)
	assert.Nil(t, err)
	entitlement := getEntitlementForTesting(resource, resourceDisplayName, roleEntitlement)
	client = getClientForTesting(apiToken, apiKey)
	roleBuilder := getRoleBuilderForTesting(client)
	_, err = roleBuilder.Grant(ctx, principal, entitlement)
	assert.Nil(t, err)
}

func TestResourceTypeRevoke(t *testing.T) {
	// --revoke-grant "role:1963e6e3aca5ac9a7a91609a0040ab02:Firewall:user:9d9a62a5b834a8c9c5cf43cd234dfd4a"
	var (
		resourceDisplayName = "Firewall Cloudflare role"
		roleEntitlement     = "Firewall"
		userEmail           = "miguel_chavez_m@hotmail.com"
		roleId              = "1963e6e3aca5ac9a7a91609a0040ab02"
		roleName            = "Firewall"
		client              *cloudflare.API
	)
	if apiKey == "" && apiToken == "" {
		t.Skip()
	}
	accUser := getAccountMemberForTesting(accountID, userId, userEmail)
	ur, err := userResource(*accUser)
	assert.Nil(t, err)
	role := getRoleForTesting(roleId, resourceDisplayName, roleEntitlement)
	resource, err := roleResource(*role, resourceTypeRole, nil)
	assert.Nil(t, err)
	client = getClientForTesting(apiToken, apiKey)
	roleBuilder := getRoleBuilderForTesting(client)
	gr := grant.NewGrant(resource, roleName, ur.Id)
	annos := annotations.Annotations(gr.Annotations)
	v1Identifier := &v2.V1Identifier{
		Id: V1GrantID(V1MembershipEntitlementID(roleId), userId),
	}
	annos.Update(v1Identifier)
	gr.Annotations = annos
	_, err = roleBuilder.Revoke(ctx, gr)
	assert.Nil(t, err)
}

func getRoleBuilderForTesting(client *cloudflare.API) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
		httpClient:   uhttp.NewBaseHttpClient(httpClient),
		accountId:    accountID,
		emailId:      emailId,
	}
}

func getAccountMemberForTesting(accountId, userId, email string) *cloudflare.AccountMember {
	return &cloudflare.AccountMember{
		ID: accountId,
		User: cloudflare.AccountMemberUserDetails{
			ID:        userId,
			FirstName: "",
			LastName:  "",
			Email:     email,
		},
	}
}

func getRoleForTesting(roleId, roleName, roleDescription string) *cloudflare.AccountRole {
	return &cloudflare.AccountRole{
		ID:          roleId,
		Name:        roleName,
		Description: roleDescription,
	}
}

func getEntitlementForTesting(resource *v2.Resource, resourceDisplayName, roleEntitlement string) *v2.Entitlement {
	options := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeRole),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resourceDisplayName, roleEntitlement)),
		ent.WithDescription(fmt.Sprintf("%s of %s Cloudflare role", roleEntitlement, resourceDisplayName)),
	}

	return ent.NewAssignmentEntitlement(resource, roleEntitlement, options...)
}

func getHttpClientForTesting() *http.Client {
	httpClient, _ := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	return httpClient
}

func getClientForTesting(apiToken, apiKey string) *cloudflare.API {
	var client *cloudflare.API
	if apiToken != "" {
		client, _ = cloudflare.NewWithAPIToken(apiToken, cloudflare.HTTPClient(httpClient))
	}

	if apiKey != "" {
		client, _ = cloudflare.New(apiKey, emailId, cloudflare.HTTPClient(httpClient))
	}

	return client
}
