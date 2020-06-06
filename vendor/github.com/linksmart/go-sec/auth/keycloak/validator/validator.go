// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

// Package validator implements OpenID Connect token validation obtained from Keycloak
package validator

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/linksmart/go-sec/auth/validator"
)

const DriverName = "keycloak"

type KeycloakValidator struct {
	publicKey *rsa.PublicKey
}

func init() {
	// Register the driver as a auth/validator
	validator.Register(DriverName, &KeycloakValidator{})
}

// Validate validates the token
func (v *KeycloakValidator) Validate(serverAddr, clientID, tokenString string) (bool, *validator.UserProfile, error) {

	if v.publicKey == nil {
		var err error
		v.publicKey, err = queryPublicKey(serverAddr)
		if err != nil {
			return false, nil, fmt.Errorf("error querying public key: %s", err)
		}
	}

	type expectedClaims struct {
		jwt.StandardClaims
		Type              string   `json:"typ"`
		PreferredUsername string   `json:"preferred_username"`
		Groups            []string `json:"groups"`
	}
	// Parse the jwt id_token
	token, err := jwt.ParseWithClaims(tokenString, &expectedClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Make sure that the algorithm is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unable to validate authentication token: unexpected signing method: %v", token.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		// Check the validation errors
		if token != nil && !token.Valid {
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					return false, &validator.UserProfile{Status: fmt.Sprintf("invalid token.")}, nil
				} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					return false, &validator.UserProfile{Status: fmt.Sprintf("token is either expired or not active yet")}, nil
				} else {
					return false, &validator.UserProfile{Status: fmt.Sprintf("error validating the token: %s", err)}, nil
				}
			} else {
				return false, &validator.UserProfile{Status: fmt.Sprintf("invalid token: %s", err)}, nil
			}
		}

		return false, &validator.UserProfile{Status: fmt.Sprintf("error parsing jwt token: %s", err)}, nil
	}

	// Validate the other claims
	claims, ok := token.Claims.(*expectedClaims)
	if !ok {
		return false, nil, fmt.Errorf("unable to extract claims from the jwt id_token")
	}
	if claims.Type != "ID" {
		return false, &validator.UserProfile{Status: fmt.Sprintf("unexpected token type: %s, expected `ID` (id_token)", claims.Type)}, nil
	}
	if claims.Audience != clientID {
		return false, &validator.UserProfile{Status: fmt.Sprintf("token is issued for another client: %s", claims.Audience)}, nil
	}
	if claims.Issuer != serverAddr {
		return false, &validator.UserProfile{Status: fmt.Sprintf("token is issued by another provider: %s", claims.Issuer)}, nil
	}

	// return user profile from claims
	return true, &validator.UserProfile{
		Username: claims.PreferredUsername,
		Groups:   claims.Groups,
	}, nil

}

func queryPublicKey(serverAddr string) (*rsa.PublicKey, error) {

	res, err := http.Get(serverAddr)
	if err != nil {
		return nil, fmt.Errorf("error getting the public key from the authentication server: %s", err)
	}
	defer res.Body.Close()

	var body struct {
		PublicKey string `json:"public_key"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return nil, fmt.Errorf("error decoding the public key from the authentication server response: %s", err)
	}

	// Decode the public key
	decoded, err := base64.StdEncoding.DecodeString(body.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding the authentication server public key: %s", err)
	}

	// Parse the public key
	parsed, err := x509.ParsePKIXPublicKey(decoded)
	if err != nil {
		return nil, fmt.Errorf("error parsing the authentication server public key: %s", err)
	}

	publicKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("the authentication server's public key type is not RSA")
	}

	return publicKey, nil
}
