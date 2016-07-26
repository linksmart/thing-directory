// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"linksmart.eu/lc/sec/auth/validator"
	"github.com/kylewolfe/simplexml"
)

const (
	oauthProfilePath        = "/oauth2.0/profile"
	casProtocolValidatePath = "/p3/serviceValidate"
	driverName              = "cas"
)

type CASValidator struct{}

var logger *log.Logger

func init() {
	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", driverName), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	// Register the driver as a auth/validator
	validator.Register(driverName, &CASValidator{})
}

// Validate Service Ticket (CAS Protocol)
func (v *CASValidator) Validate(serverAddr, serviceID, ticket string) (bool, *validator.UserProfile, error) {
	res, err := http.Get(fmt.Sprintf("%s%s?service=%s&ticket=%s", serverAddr, casProtocolValidatePath, serviceID, ticket))
	if err != nil {
		return false, nil, fmt.Errorf("%s", err)
	}

	// Check for server errors
	if res.StatusCode != http.StatusOK {
		return false, nil, fmt.Errorf(res.Status)
	}

	// User attributes / error message
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return false, nil, fmt.Errorf("%s", err)
	}

	// Create an xml document from response body
	doc, err := simplexml.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return false, nil, fmt.Errorf("Unexpected error while validating service token.")
	}

	var profile validator.UserProfile
	// StatusCode is 200 for all responses (valid, expired, missing)
	// Check if response contains authenticationSuccess tag
	success := doc.Root().Search().ByName("authenticationSuccess").One()
	// There is no authenticationSuccess tag
	// Token is invalid or there are response errors
	if success == nil {
		// Check if response contains authenticationFailure tag
		failure := doc.Root().Search().ByName("authenticationFailure").One()
		if failure == nil {
			return false, nil, fmt.Errorf("Unexpected error while validating service token.")
		}
		// Extract the error message
		errMsg, err := failure.Value()
		if err != nil {
			return false, nil, fmt.Errorf("Unexpected error. No error message.")
		}
		profile.Status = strings.TrimSpace(errMsg)
		return false, &profile, nil
	}
	// Token is valid
	logger.Println("Validate()", res.Status, "Valid ticket.")

	// Extract username
	userTag := doc.Root().Search().ByName("authenticationSuccess").ByName("user").One()
	if userTag == nil {
		return false, nil, fmt.Errorf("Could not find `user` from validation response.")
	}
	user, err := userTag.Value()
	if err != nil {
		return false, nil, fmt.Errorf("Could not get value of `user` from validation response.")
	}
	// NOTE:
	// temporary workaround until CAS bug is fixed
	ldapDescription := strings.Split(user, "-")
	if len(ldapDescription) == 2 {
		profile.Username = ldapDescription[0]
		profile.Groups = append(profile.Groups, ldapDescription[1])
	} else if len(ldapDescription) == 1 {
		profile.Username = ldapDescription[0]
	} else {
		return false, nil, fmt.Errorf("Unexpected format for `user` in validation response.")
	}

	// Valid token + attributes
	return true, &profile, nil
}

// Validate Service Token (OAUTH)
//func (ca *CASValidator) ValidateServiceToken(serviceToken string) (bool, map[string]interface{}, error) {
//	fmt.Println("CAS: Validating Service Token...")

//	var bodyMap map[string]interface{}
//	res, err := http.Get(fmt.Sprintf("%s%s?access_token=%s", ca.conf.CasServer, oauthProfilePath, serviceToken))
//	if err != nil {
//		return false, bodyMap, fErr(err)
//	}
//	fmt.Println("CAS:", res.Status)

//	// Check for server errors
//	if res.StatusCode != http.StatusOK {
//		return false, bodyMap, fErr(fmt.Errorf(res.Status))
//	}

//	// User attributes / error message
//	body, err := ioutil.ReadAll(res.Body)
//	defer res.Body.Close()
//	if err != nil {
//		return false, bodyMap, fErr(err)
//	}
//	res.Body.Close()

//	if len(body) == 0 { // body is empty due to CAS bug
//		fmt.Println("CAS: Token was valid.")
//		return true, bodyMap, nil
//	}

//	err = json.Unmarshal(body, &bodyMap)
//	if err != nil {
//		return false, bodyMap, fErr(err)
//	}
//	// StatusCode is 200 for all responses (valid, expired, missing)
//	// Check the error message
//	errMsg, ok := bodyMap["error"]
//	if ok {
//		fmt.Println("CAS: Error:", errMsg)
//		return false, bodyMap, nil
//	} else {
//		fmt.Println("CAS: Token was valid.")
//	}

//	// Valid token + attributes
//	return true, bodyMap, nil
//}
