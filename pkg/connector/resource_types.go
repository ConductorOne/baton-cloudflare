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
		Annotations: v1AnnotationsWithPermissions("user", capabilityPermissions(
			"Access: Organizations, Identity Providers and Groups:Read",
			"Account Settings: Read",
		)),
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
		Annotations: v1AnnotationsWithPermissions("role", capabilityPermissions(
			"Access: Organizations, Identity Providers and Groups:Read",
			"Account Settings: Read",
			"Account Settings: Edit",
		)),
	}
)
