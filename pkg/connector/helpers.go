package connector

import (
	"fmt"
	"strconv"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
)

const (
	MembershipEntitlementIDTemplate = "membership:%s"
	V1GrantIDTemplate               = "grant:%s:%s"
)

func v1AnnotationsForResourceType(resourceTypeID string) annotations.Annotations {
	annos := annotations.Annotations{}
	annos.Update(&v2.V1Identifier{
		Id: resourceTypeID,
	})
	return annos
}

func V1MembershipEntitlementID(resourceID string) string {
	return fmt.Sprintf(MembershipEntitlementIDTemplate, resourceID)
}

func V1GrantID(entitlementID string, userID string) string {
	return fmt.Sprintf(V1GrantIDTemplate, entitlementID, userID)
}

func convertPageToken(token string) (int, error) {
	if token == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(token)
	return page, err
}

func convertNextPageToken(token int, responseLength int) string {
	if responseLength == 0 {
		return ""
	}
	strToken := strconv.FormatInt(int64(token)+1, 10)
	return strToken
}
