// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package validator implements OpenID Connect token validation obtained from Keycloak
package validator

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/linksmart/go-sec/auth/validator"
)

const DriverName = "keycloak"

type KeycloakValidator struct{}

var (
	logger    *log.Logger
	publicKey *rsa.PublicKey
)

func init() {
	// Initialize the logger
	logger = log.New(os.Stdout, fmt.Sprintf("[%s] ", DriverName), 0)
	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}

	// Register the driver as a auth/validator
	validator.Register(DriverName, &KeycloakValidator{})
}

// Validate validates the token
func (v *KeycloakValidator) Validate(serverAddr, clientID, ticket string) (bool, *validator.UserProfile, error) {

	if publicKey == nil {
		// Get the public key
		res, err := http.Get(serverAddr)
		if err != nil {
			return false, nil, fmt.Errorf("error getting the public key from the authentication server: %s", err)
		}
		defer res.Body.Close()

		var body struct {
			PublicKey string `json:"public_key"`
		}
		err = json.NewDecoder(res.Body).Decode(&body)
		if err != nil {
			return false, nil, fmt.Errorf("error getting the public key from the authentication server response: %s", err)
		}

		// Decode the public key
		decoded, err := base64.StdEncoding.DecodeString(body.PublicKey)
		if err != nil {
			return false, nil, fmt.Errorf("error decoding the authentication server public key: %s", err)
		}

		// Parse the public key
		parsed, err := x509.ParsePKIXPublicKey(decoded)
		if err != nil {
			return false, nil, fmt.Errorf("error pasring the authentication server public key: %s", err)
		}

		var ok bool
		if publicKey, ok = parsed.(*rsa.PublicKey); !ok {
			return false, nil, fmt.Errorf("the authentication server's public key type is not RSA")
		}
	}

	// Parse the jwt id_token
	token, err := jwt.Parse(ticket, func(token *jwt.Token) (interface{}, error) {
		// Make sure that the algorithm is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unable to validate authentication token. Unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return false, &validator.UserProfile{Status: fmt.Sprintf("Invalid token: %s", err)}, nil
	}

	// Check the validation errors
	if !token.Valid {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return false, &validator.UserProfile{Status: fmt.Sprintf("Invalid token.")}, nil
			} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
				return false, &validator.UserProfile{Status: fmt.Sprintf("Token is either expired or not active yet")}, nil
			} else {
				return false, &validator.UserProfile{Status: fmt.Sprintf("Error validating the token: %s", err)}, nil
			}
		} else {
			return false, &validator.UserProfile{Status: fmt.Sprintf("Invalid token: %s", err)}, nil
		}
	}

	// Get the claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if claims["typ"].(string) != "ID" {
			return false, &validator.UserProfile{Status: fmt.Sprintf("Wrong token type `%s` for accessing resource. Expecting type `ID`.", claims["typ"])}, nil
		}
		// Check if audience matches the client id
		if claims["aud"].(string) != clientID {
			return false, &validator.UserProfile{Status: fmt.Sprintf("The token is issued for client `%s` rather than `%s`.", claims["aud"], clientID)}, nil
		}

		// Get the user data
		groupInts, ok := claims["groups"].([]interface{})
		if !ok {
			return false, nil, fmt.Errorf("unable to get the user's group membership")
		}
		// convert []interface{} to []string
		groups := make([]string, len(groupInts))
		for i := range groupInts {
			groups[i] = groupInts[i].(string)
		}
		username, ok := claims["preferred_username"].(string)
		if !ok {
			return false, nil, fmt.Errorf("unable to get the user's username")
		}
		return true, &validator.UserProfile{
			Username: username,
			Groups:   groups,
		}, nil
	}
	return false, nil, fmt.Errorf("unable to extract claims from the jwt id_token")
}
