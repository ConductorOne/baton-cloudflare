package connector

import (
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: buildAnnotations(
			&v2.V1Identifier{Id: "user"},
			capabilityPermissions(
				"Access: Organizations, Identity Providers and Groups:Read",
				"Account Settings: Read",
				"Account Settings: Edit",
			),
			&v2.SkipEntitlementsAndGrants{},
		),
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
		Annotations: buildAnnotations(
			&v2.V1Identifier{Id: "role"},
			capabilityPermissions(
				"Access: Organizations, Identity Providers and Groups:Read",
				"Account Settings: Read",
				"Account Settings: Edit",
			),
		),
	}
	resourceTypeAPIToken = &v2.ResourceType{
		Id:          "api_token",
		DisplayName: "API Token",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_SECRET,
		},
		Annotations: buildAnnotations(
			&v2.V1Identifier{Id: "api_token"},
			capabilityPermissions(
				"Account API Tokens:Read",
			),
			&v2.SkipEntitlementsAndGrants{},
		),
	}
)
