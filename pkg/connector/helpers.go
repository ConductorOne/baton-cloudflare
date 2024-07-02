package connector

import (
	"errors"
	"fmt"
	"strconv"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

const (
	MembershipEntitlementIDTemplate = "membership:%s"
	V1GrantIDTemplate               = "grant:%s:%s"
)

type CloudflareError struct {
	ErrorMessage     string                   `json:"error"`
	ErrorDescription string                   `json:"error_description"`
	ErrorCode        int                      `json:"errorCode,omitempty"`
	ErrorSummary     string                   `json:"errorSummary,omitempty" toml:"error_description"`
	ErrorLink        string                   `json:"errorLink,omitempty"`
	ErrorId          string                   `json:"errorId,omitempty"`
	ErrorCauses      []map[string]interface{} `json:"errorCauses,omitempty"`
}

func (b *CloudflareError) Error() string {
	return b.ErrorMessage
}

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

func getError(err error) error {
	var bitbucketErr *CloudflareError
	if err == nil {
		return nil
	}

	if errors.As(err, &bitbucketErr) {
		return fmt.Errorf("%s %s", bitbucketErr.Error(), bitbucketErr.ErrorSummary)
	}

	return err
}

func WithAuthorizationBearerHeader(token string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", "Bearer "+token)
}
