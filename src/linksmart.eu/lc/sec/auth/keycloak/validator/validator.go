// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"linksmart.eu/lc/sec/auth/validator"
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
func (v *KeycloakValidator) Validate(serverAddr, clientID, ticket string) (bool, *validator.UserProfile, error) {

	// Get the public key
	res, err := http.Get(serverAddr)
	if err != nil {
		return false, nil, fmt.Errorf("Error getting the public key from the authentication server: %s", err)
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return false, nil, fmt.Errorf("Error getting the public key from the authentication server response: %s", err)
	}

	body := make(map[string]interface{})
	err = json.Unmarshal(b, &body)
	if err != nil {
		return false, nil, fmt.Errorf("Error getting the public key from the authentication server response: %s", err)
	}

	// Decode the public key
	decoded, err := base64.StdEncoding.DecodeString(body["public_key"].(string))
	if err != nil {
		return false, nil, fmt.Errorf("Error decoding the authentication server public key: %s", err)
	}

	// Parse the public key
	publicKey, err := x509.ParsePKIXPublicKey(decoded)
	if err != nil {
		return false, nil, fmt.Errorf("Error pasring the authentication server public key: %s", err)
	}

	if _, ok := publicKey.(*rsa.PublicKey); !ok {
		return false, nil, fmt.Errorf("The authentication server's public key type is not RSA.")
	}

	// Parse the jwt id_token
	token, err := jwt.Parse(ticket, func(token *jwt.Token) (interface{}, error) {
		// Make sure that the algorithm is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unable to validate authentication token. Unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	// Check the validation errors
	if !token.Valid {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return false, &validator.UserProfile{Status: fmt.Sprintf("Invalid authentication token.")}, nil
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				return false, &validator.UserProfile{Status: fmt.Sprintf("Authentication token is either expired or not active yet")}, nil
			} else {
				return false, &validator.UserProfile{Status: fmt.Sprintf("Error validating the authentication token: %s", err)}, nil
			}
		} else {
			return false, &validator.UserProfile{Status: fmt.Sprintf("Invalid authentication token: %s", err)}, nil
		}
	}

	// Get the claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// Check if audience matches the client id
		if claims["aud"].(string) != clientID {
			return false, &validator.UserProfile{Status: fmt.Sprintf("Unable to authenticate with client `%s`. Expecting `%s`.", claims["aud"].(string), clientID)}, nil
		}

		// Get the user data
		// convert groups attribute to []string
		groupInts := claims["groups"].([]interface{})
		groups := make([]string, len(groupInts))
		for i := range groupInts {
			groups[i] = groupInts[i].(string)
		}
		return true, &validator.UserProfile{
			Username: claims["preferred_username"].(string),
			Groups:   groups,
		}, nil
	}
	return false, nil, fmt.Errorf("Unable to extract claims from the jwt token.")
}
