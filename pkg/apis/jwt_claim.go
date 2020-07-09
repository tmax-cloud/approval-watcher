package apis

import "github.com/dgrijalva/jwt-go"

type JwtClaim struct {
	TokenId string `json:"tokenId"`
	Id      string `json:"id"`
	jwt.StandardClaims
}
