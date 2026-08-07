package connector

import (
	"testing"

	"github.com/cloudflare/cloudflare-go"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserResourceStatus(t *testing.T) {
	for _, tc := range []struct {
		name           string
		memberStatus   string
		expectedStatus v2.Status_ResourceStatus
		expectedDetail string
	}{
		{"accepted", "accepted", v2.Status_RESOURCE_STATUS_ENABLED, "Accepted"},
		{"unknown", "rejected", v2.Status_RESOURCE_STATUS_DISABLED, "Rejected"},
		{"empty", "", v2.Status_RESOURCE_STATUS_DISABLED, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			member := cloudflare.AccountMember{
				ID:     "member-1",
				Status: tc.memberStatus,
				User: cloudflare.AccountMemberUserDetails{
					ID:        "user-1",
					Email:     "someone@example.com",
					FirstName: "Some",
					LastName:  "One",
				},
			}

			resource, err := userResource(member)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, resource.GetStatus().GetStatus())
			assert.Equal(t, tc.expectedDetail, resource.GetStatus().GetDetails())
		})
	}
}

// The member ID is read back out of the resource-level profile during Grant/Revoke,
// so it has to survive the move off the user trait.
func TestUserResourceProfile(t *testing.T) {
	member := cloudflare.AccountMember{
		ID:     "member-1",
		Status: "accepted",
		User: cloudflare.AccountMemberUserDetails{
			ID:        "user-1",
			Email:     "someone@example.com",
			FirstName: "Some",
			LastName:  "One",
		},
	}

	resource, err := userResource(member)
	require.NoError(t, err)

	memberID, found := rs.GetProfileStringValue(resource.GetProfile(), memberIdProfileKey)
	require.True(t, found, "expected %s on the resource profile", memberIdProfileKey)
	assert.Equal(t, member.ID, memberID)

	email, found := rs.GetProfileStringValue(resource.GetProfile(), "email")
	require.True(t, found)
	assert.Equal(t, member.User.Email, email)
}
