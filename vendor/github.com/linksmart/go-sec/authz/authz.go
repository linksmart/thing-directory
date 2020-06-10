// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package authz provides simple rule-based authorization that can be used to implement access control
package authz

import (
	"strings"
)

// GroupAnonymous is the group name for unauthenticated users
const GroupAnonymous = "anonymous"

// Authorized checks whether a user/group is authorized to access a resource using the specific method
func (authz *Conf) Authorized(resource, method string, claims *Claims) bool {
	if claims == nil {
		claims = &Claims{Groups: []string{GroupAnonymous}}
	}
	// Create a tree of paths
	// e.g. /path1/path2/path3 -> [/path1/path2/path3 /path1/path2 /path1]
	// e.g. / -> [/]
	resourceSplit := strings.Split(resource, "/")[1:] // split and drop the first part (empty string before slash)
	resourceTree := make([]string, 0, len(resourceSplit))
	// construct tree from longest to shortest (/path1) path
	for i := len(resourceSplit); i >= 1; i-- {
		resourceTree = append(resourceTree, "/"+strings.Join(resourceSplit[:i], "/"))
	}
	//fmt.Printf("%s -> %v -> %v\n", resource, resourceSplit, resourceTree)

	for _, rule := range authz.Rules {
		for _, res := range resourceTree {
			// Return true if user or group matches a rule
			if inSlice(res, rule.Resources) &&
				inSlice(method, rule.Methods) &&
				(inSlice(claims.Username, rule.Users) || hasIntersection(claims.Groups, rule.Groups) || inSlice(claims.ClientID, rule.Clients)) {
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
