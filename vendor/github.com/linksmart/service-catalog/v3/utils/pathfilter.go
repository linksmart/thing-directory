// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	FOpEquals   = "equals"
	FOpPrefix   = "prefix"
	FOpSuffix   = "suffix"
	FOpContains = "contains"
)

func recursiveMatch(data interface{}, path []string) interface{} {
	// path matched. return the value
	if len(path) == 0 {
		return data
	}

	// match path recursively
	switch data.(type) {
	case map[string]interface{}:
		for k, v := range data.(map[string]interface{}) {
			if k == path[0] {
				// logger.Printf("MAP key: %s, path: %s, value: %v", k, path, v)
				return recursiveMatch(v, path[1:])
			}
		}
	case []interface{}:
		for _, v := range data.([]interface{}) {
			// follow the array's elements
			if _, ok := v.(map[string]interface{})[path[0]]; ok {
				// logger.Printf("ARRAY key: %s, path: %s, value: %v", path[0], path, v)
				return recursiveMatch(v, path)
			}
		}
	default:
		logger.Println("Unknown type for", data)
	}

	return nil
}

func MatchObject(object interface{}, path []string, op string, value string) (bool, error) {
	var m interface{}
	b, err := json.Marshal(object)
	if err != nil {
		return false, errors.New("unable to parse object into JSON")
	}
	json.Unmarshal(b, &m)

	// check if the path exists
	v := recursiveMatch(m, path)
	if v == nil {
		return false, nil
	}

	// convert everything to lower-case string
	stringValue := strings.ToLower(fmt.Sprint(v))
	value = strings.ToLower(value)

	switch op {
	case FOpEquals:
		if stringValue == value {
			return true, nil
		} else {
			return false, nil
		}
	case FOpPrefix:
		if strings.HasPrefix(stringValue, value) {
			return true, nil
		} else {
			return false, nil
		}
	case FOpSuffix:
		if strings.HasSuffix(stringValue, value) {
			return true, nil
		} else {
			return false, nil
		}
	case FOpContains:
		if strings.Contains(stringValue, value) {
			return true, nil
		} else {
			return false, nil
		}
	}
	return false, fmt.Errorf("unknown filter operation: %s. Should be either of %v", op,
		strings.Join([]string{FOpEquals, FOpPrefix, FOpSuffix, FOpContains}, ", "))
}
