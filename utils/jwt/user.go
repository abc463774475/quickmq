package jwt

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type UserClaims struct {
	UserName string
	jwt.StandardClaims
}

func GetUserToken(userName string, duration time.Duration) (string, error) {
	uClaims := &UserClaims{
		UserName: userName,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(duration).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, uClaims)
	s, e := token.SignedString(rsaPrivateKey)
	return s, e
}

func ParseUserToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexept singing methods %v", token.Header["alg"])
			}
			return rsaPublicKey, nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, ErrNotTheUserClaims
	}

	if !token.Valid {
		return nil, ErrTokenExpired
	}

	return claims, nil
}
