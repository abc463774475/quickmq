package jwt

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type RoleClaims struct {
	UserName     string
	SelectRoleID int64
	GSID         int64
	Name         string
	Sex          int
	Career       int
	jwt.StandardClaims
}

func GetRoleToken(userName string, sRoleID int64,
	gsID int64, name string, sex, career int, duration time.Duration,
) (string, error) {
	uClaims := &RoleClaims{
		UserName:     userName,
		SelectRoleID: sRoleID,
		GSID:         gsID,
		Name:         name,
		Sex:          sex,
		Career:       career,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(duration).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, uClaims)
	s, e := token.SignedString(rsaPrivateKey)
	return s, e
}

func ParseRoleToken(tokenStr string) (*RoleClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &RoleClaims{},
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexept singing methods %v", token.Header["alg"])
			}
			return rsaPublicKey, nil
		})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*RoleClaims)
	if !ok {
		return nil, ErrNotTheUserClaims
	}

	if !token.Valid {
		return nil, ErrTokenExpired
	}

	return claims, nil
}
