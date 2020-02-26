// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package utils

import (
	"fmt"
	"strconv"
)

const (
	GetParamPage    = "page"
	GetParamPerPage = "per_page"
)

// Returns a 'slice' of the given slice based on the requested 'page'
func GetPageOfSlice(slice []string, page, perPage, maxPerPage int) ([]string, error) {
	err := ValidatePagingParams(page, perPage, maxPerPage)
	if err != nil {
		return nil, err
	}

	keys := []string{}
	if page == 1 {
		// first page
		if perPage > len(slice) {
			keys = slice
		} else {
			keys = slice[:perPage]
		}
	} else if page == int(len(slice)/perPage)+1 {
		// last page
		keys = slice[perPage*(page-1):]

	} else if page <= len(slice)/perPage && page*perPage <= len(slice) {
		// slice
		r := page * perPage
		l := r - perPage
		keys = slice[l:r]
	}
	return keys, nil
}

// Returns offset and limit representing a subset of the given slice total size
//	 based on the requested 'page'
func GetPagingAttr(total, page, perPage, maxPerPage int) (int, int, error) {
	err := ValidatePagingParams(page, perPage, maxPerPage)
	if err != nil {
		return 0, 0, err
	}

	if page == 1 {
		// first page
		if perPage > total {
			return 0, total, nil
		} else {
			return 0, perPage, nil
		}
	} else if page == int(total/perPage)+1 {
		// last page
		return perPage * (page - 1), total - perPage*(page-1), nil
	} else if page <= total/perPage && page*perPage <= total {
		// another page
		r := page * perPage
		l := r - perPage
		return l, r - l, nil
	}
	return 0, 0, nil
}

// Validates paging parameters
func ValidatePagingParams(page, perPage, maxPerPage int) error {
	if page < 1 {
		return fmt.Errorf("%s parameter must be positive", GetParamPage)
	}
	if perPage < 1 {
		return fmt.Errorf("%s parameter must be positive", GetParamPerPage)
	}
	if perPage > maxPerPage {
		return fmt.Errorf("%s must less than or equal to %d", GetParamPerPage, maxPerPage)
	}
	return nil
}

// Parses string paging parameters to integers
func ParsePagingParams(page, perPage string, maxPerPage int) (int, int, error) {
	var parsedPage, parsedPerPage int
	var err error

	if page == "" {
		parsedPage = 1
	} else {
		parsedPage, err = strconv.Atoi(page)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid value for parameter %s: %s", GetParamPage, page)
		}
	}

	if perPage == "" {
		parsedPerPage = maxPerPage
	} else {
		parsedPerPage, err = strconv.Atoi(perPage)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid value for parameter %s: %s", GetParamPerPage, perPage)
		}
	}

	return parsedPage, parsedPerPage, ValidatePagingParams(parsedPage, parsedPerPage, maxPerPage)
}
