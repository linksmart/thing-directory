// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package authz

import "errors"

// Authorization struct
type Conf struct {
	// Enabled toggles authorization
	Enabled bool `json:"enabled"`
	// Authorization rules
	Rules []Rule `json:"rules"`
}

// Authorization rule
type Rule struct {
	Resources []string `json:"resources"`
	Methods   []string `json:"methods"`
	Users     []string `json:"users"`
	Groups    []string `json:"groups"`
}

// Validate authorization config
func (authz *Conf) Validate() error {

	// Check each authorization rule
	for _, rule := range authz.Rules {
		if len(rule.Resources) == 0 {
			return errors.New("no resources in an authorization rule")
		}
		if len(rule.Methods) == 0 {
			return errors.New("no methods in an authorization rule")
		}
		if len(rule.Users) == 0 && len(rule.Groups) == 0 {
			return errors.New("at least one user or group must be assigned to each authorization rule")
		}
	}

	return nil
}
