package handlers

import (
	"errors"
	"fmt"
	"policy-server/models"
)

type Validator struct{}

func (v *Validator) ValidatePolicies(policies []models.Policy) error {
	if len(policies) == 0 {
		return errors.New("missing policies")
	}

	for _, policy := range policies {
		if policy.Source.ID == "" {
			return errors.New("missing source id")
		}
		if policy.Destination.ID == "" {
			return errors.New("missing destination id")
		}
		if policy.Destination.Protocol != "udp" && policy.Destination.Protocol != "tcp" {
			return errors.New("invalid destination protocol, specify either udp or tcp")
		}
		if policy.Destination.Port < 1 || policy.Destination.Port > 65535 {
			return fmt.Errorf("invalid destination port value %d, must be 1-65535", policy.Destination.Port)
		}

		if policy.Source.Tag != "" || policy.Destination.Tag != "" {
			return errors.New("tags may not be specified")
		}
	}
	return nil
}
