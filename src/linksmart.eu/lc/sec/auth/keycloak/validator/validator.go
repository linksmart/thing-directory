// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"linksmart.eu/lc/sec/auth/validator"
	"log"
	"strconv"
)

const (
	userInfoPath = "/protocol/openid-connect/userinfo"
	driverName   = "keycloak"
)

type KeycloakValidator struct{}

var logger *log.Logger

func init() {
	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", driverName), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	// Register the driver as a auth/validator
	validator.Register(driverName, &KeycloakValidator{})
}

// Validate validates the token
func (v *KeycloakValidator) Validate(serverAddr, serviceID, ticket string) (bool, *validator.UserProfile, error) {

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", serverAddr, userInfoPath), nil)
	if err != nil {
		return false, nil, fmt.Errorf("%s", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", ticket))
	res, err := client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("%s", err)
	}

	if res.StatusCode == http.StatusForbidden {
		return false, nil, nil
	}

	// Check for server errors
	if res.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf("%s", res.Status)
	}
	logger.Println("Validate()", res.Status, "Valid ticket.")

	// User attributes / error message
	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return false, nil, fmt.Errorf("%s", err)
	}

	var body struct {
		Groups   []string `json:"groups"`
		Username string   `json:"preferred_username"`
	}
	err = json.Unmarshal(b, &body)
	if err != nil {
		return false, nil, fmt.Errorf("Unable to parse response body: %s", err)
	}

	if body.Username == "" && len(body.Groups) == 0 {
		return false, nil, fmt.Errorf("User profile does not contain `preferred_username` and `groups`.")
	}

	// Valid token + attributes
	return true, &validator.UserProfile{
		Username: body.Username,
		Groups:   body.Groups,
	}, nil
}
