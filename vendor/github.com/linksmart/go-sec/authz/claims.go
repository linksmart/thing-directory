package authz

// Claims are the profile attributes of user/client that are part of the JWT claims
type Claims struct {
	Username string
	Groups   []string
	ClientID string // for tokens issued as part of client credentials grant
	// Status is the message given when token is not validated
	Status string
}
