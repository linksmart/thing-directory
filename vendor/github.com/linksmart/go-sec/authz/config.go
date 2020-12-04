// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package authz

import (
	"errors"
	"fmt"
)

// Authorization struct
type Conf struct {
	// Enabled toggles authorization
	Enabled bool `json:"enabled"`
	// Authorization rules
	Rules Rules `json:"rules"`
}

type Rules []Rule

// Authorization rule
type Rule struct {
	Paths                  []string `json:"paths"`
	Methods                []string `json:"methods"`
	Users                  []string `json:"users"`
	Groups                 []string `json:"groups"`
	Roles                  []string `json:"roles"`
	Clients                []string `json:"clients"`
	ExcludePathSubstrtings []string `json:"excludePathSubstrings"`
	// Deprecated. Use Paths instead.
	Resources []string `json:"resources"`
	// Deprecated. Use ExcludePathSubstrtings instead.
	DenyPathSubstrtings []string `json:"denyPathSubstrings"`
}

// Validate authorization config
func (authz Conf) Validate() error {

	// Check each authorization rule
	for _, rule := range authz.Rules {
		// take Paths from deprecated Resources
		if len(rule.Paths) == 0 && len(rule.Resources) != 0 {
			fmt.Println("go-sec/authz: rules.resources config is deprecated. Use rules.paths instead.")
			rule.Paths = rule.Resources
		}

		if len(rule.Paths) == 0 {
			return errors.New("no paths in an authorization rule")
		}
		if len(rule.Methods) == 0 {
			return errors.New("no methods in an authorization rule")
		}
		if len(rule.Users)+len(rule.Groups)+len(rule.Roles)+len(rule.Clients) == 0 {
			return errors.New("at least one user, group, role, or client must be set in each authorization rule")
		}

		if len(rule.DenyPathSubstrtings) != 0 {
			fmt.Println("go-sec/authz: rules.denyPathSubstrings config is deprecated. Use rules.excludePathSubstrings instead.")
		}
	}

	return nil
}
