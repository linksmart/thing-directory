// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package authz provides simple rule-based authorization that can be used to implement access control
package authz

import (
	"strings"
)

// GroupAnonymous is the group name for unauthenticated users
const GroupAnonymous = "anonymous"

// Authorized checks whether a request is authorized given the path, method, and claims
func (rules Rules) Authorized(path, method string, claims *Claims) bool {
	if claims == nil {
		claims = &Claims{Groups: []string{GroupAnonymous}}
	}
	// Create a tree of paths
	// e.g. /path1/path2/path3 -> [/path1/path2/path3 /path1/path2 /path1]
	// e.g. / -> [/]
	pathSplit := strings.Split(path, "/")[1:] // split and drop the first part (empty string before slash)
	pathTree := make([]string, 0, len(pathSplit))
	// construct tree from longest to shortest (/path1) path
	for i := len(pathSplit); i >= 1; i-- {
		pathTree = append(pathTree, "/"+strings.Join(pathSplit[:i], "/"))
	}
	//fmt.Printf("%s -> %v -> %v\n", path, pathSplit, pathTree)

	for _, rule := range rules {
		// take Paths from deprecated Resources
		if len(rule.Paths) == 0 && len(rule.Resources) != 0 {
			rule.Paths = rule.Resources
		}
		// take exclusion substrings from deprecated DenyPathSubstrtings
		if len(rule.ExcludePathSubstrtings) == 0 && len(rule.DenyPathSubstrtings) != 0 {
			rule.ExcludePathSubstrtings = rule.DenyPathSubstrtings
		}

		var excludedPath bool
		for _, substr := range rule.ExcludePathSubstrtings {
			if strings.Contains(path, substr) {
				excludedPath = true
				break
			}
		}

		for _, p := range pathTree {
			// Return true if a rule matches
			if inSlice(p, rule.Paths) &&
				inSlice(method, rule.Methods) &&
				(inSlice(claims.Username, rule.Users) ||
					hasIntersection(claims.Groups, rule.Groups) ||
					hasIntersection(claims.Roles, rule.Roles) ||
					inSlice(claims.ClientID, rule.Clients)) &&
				!excludedPath {
				return true
			}
		}
	}
	return false
}

// inSlice check whether a is in slice
func inSlice(a string, slice []string) bool {
	for _, b := range slice {
		if b == a {
			return true
		}
	}
	return false
}

// hasIntersection checks whether there is a match between two slices
func hasIntersection(slice1 []string, slice2 []string) bool {
	for _, a := range slice1 {
		for _, b := range slice2 {
			if b == a {
				return true
			}
		}
	}
	return false
}
