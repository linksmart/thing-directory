// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package authz provides simple rule-based authorization that can be used to implement access control
package authz

import "strings"

// Authorized checks whether a user/group is authorized to access resource using a specific method
func (authz *Conf) Authorized(resource, method, user string, groups []string) bool {
	// Create a tree of paths
	// e.g. parses /path1/path2/path3 to [/path1/path2/path3 /path1/path2 /path1]
	// e.g. parses / to [/]
	resource_split := strings.Split(resource, "/")
	resource_split = resource_split[1:len(resource_split)] // truncate the first slash
	var resource_tree []string
	// construct tree from longest to shortest (/path1) path
	for i := len(resource_split); i >= 1; i-- {
		resource_tree = append(resource_tree, "/"+strings.Join(resource_split[0:i], "/"))
	}
	//fmt.Println(len(resource_split), resource_split)
	//fmt.Println(len(resource_tree), resource_tree)

	// Check whether a is in slice
	inSlice := func(a string, slice []string) bool {
		for _, b := range slice {
			if b == a {
				return true
			}
		}
		return false
	}
	// Check whether there is a match between two slices
	inSliceM := func(slice1 []string, slice2 []string) bool {
		for _, a := range slice1 {
			for _, b := range slice2 {
				if b == a {
					return true
				}
			}
		}
		return false
	}

	for _, rule := range authz.Rules {
		for _, res := range resource_tree {
			// Return true if user or group matches a rule
			if inSlice(res, rule.Resources) && inSlice(method, rule.Methods) &&
				(inSlice(user, rule.Users) || inSliceM(groups, rule.Groups)) {
				return true
			}
		}
	}
	return false
}
