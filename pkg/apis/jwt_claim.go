package apis

import "github.com/dgrijalva/jwt-go"

type JwtClaim struct {
	TokenId    string `json:"tokenId"`
	Id         string `json:"id"`                 // Non-keycloak id
	KeyCloakId string `json:"preferred_username"` // Keycloak id
	jwt.StandardClaims
}
