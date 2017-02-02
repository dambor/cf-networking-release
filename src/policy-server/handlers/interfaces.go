package handlers

import (
	"net/http"
	"policy-server/models"
	"policy-server/uaa_client"
)

//go:generate counterfeiter -o fakes/http_handler.go --fake-name HTTPHandler . http_handler
type http_handler interface {
	http.Handler
}

//go:generate counterfeiter -o fakes/authenticated_handler.go --fake-name AuthenticatedHandler . authenticatedHandler
type authenticatedHandler interface {
	ServeHTTP(response http.ResponseWriter, request *http.Request, tokenData uaa_client.CheckTokenResponse)
}

//go:generate counterfeiter -o fakes/policy_cleaner.go --fake-name PolicyCleaner . policyCleaner
type policyCleaner interface {
	DeleteStalePolicies() ([]models.Policy, error)
}

//go:generate counterfeiter -o fakes/policy_guard.go --fake-name PolicyGuard . policyGuard
type policyGuard interface {
	CheckAccess(policies []models.Policy, tokenData uaa_client.CheckTokenResponse) (bool, error)
}

//go:generate counterfeiter -o fakes/policy_filter.go --fake-name PolicyFilter . policyFilter
type policyFilter interface {
	FilterPolicies(policies []models.Policy, userToken uaa_client.CheckTokenResponse) ([]models.Policy, error)
}

//go:generate counterfeiter -o fakes/store.go --fake-name Store . store
type store interface {
	All() ([]models.Policy, error)
	Create([]models.Policy) error
	Delete([]models.Policy) error
	Tags() ([]models.Tag, error)
}

//go:generate counterfeiter -o fakes/uua_client.go --fake-name UAAClient . uaaClient
type uaaClient interface {
	GetToken() (string, error)
	CheckToken(string) (uaa_client.CheckTokenResponse, error)
}

//go:generate counterfeiter -o fakes/cc_client.go --fake-name CCClient . ccClient
type ccClient interface {
	GetAppSpaces(token string, appGUIDs []string) (map[string]string, error)
	GetSpace(token, spaceGUID string) (*models.Space, error)
	GetSpaceGUIDs(token string, appGUIDs []string) ([]string, error)
	GetUserSpace(token, userGUID string, spaces models.Space) (*models.Space, error)
	GetUserSpaces(token, userGUID string) (map[string]struct{}, error)
}

//go:generate counterfeiter -o fakes/validator.go --fake-name Validator . validator
type validator interface {
	ValidatePolicies(policies []models.Policy) error
}
