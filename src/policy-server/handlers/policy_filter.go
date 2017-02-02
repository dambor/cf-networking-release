package handlers

import (
	"fmt"
	"policy-server/models"
	"policy-server/uaa_client"
)

type PolicyFilter struct {
	CCClient  ccClient
	UAAClient uaaClient
}

func (g *PolicyFilter) FilterPolicies(policies []models.Policy, userToken uaa_client.CheckTokenResponse) ([]models.Policy, error) {
	for _, scope := range userToken.Scope {
		if scope == "network.admin" {
			return policies, nil
		}
	}

	token, err := g.UAAClient.GetToken()
	if err != nil {
		return nil, fmt.Errorf("getting token: %s", err)
	}

	appGuids := uniqueAppGUIDs(policies)
	appSpaces, err := g.CCClient.GetAppSpaces(token, appGuids)
	if err != nil {
		return nil, fmt.Errorf("getting app spaces: %s", err)
	}

	userSpaces, err := g.CCClient.GetUserSpaces(token, userToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("getting user spaces: %s", err)
	}

	filtered := filter(policies, appSpaces, userSpaces)

	return filtered, nil
}

func filter(policies []models.Policy, appSpaces map[string]string, userSpaces map[string]struct{}) []models.Policy {
	filtered := []models.Policy{}

	for _, policy := range policies {
		_, sourceFound := userSpaces[appSpaces[policy.Source.ID]]
		_, destFound := userSpaces[appSpaces[policy.Destination.ID]]
		if sourceFound && destFound {
			filtered = append(filtered, policy)
		}
	}
	return filtered
}
