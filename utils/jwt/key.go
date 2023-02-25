package jwt

import (
	"crypto/rsa"
	_ "embed"

	"github.com/dgrijalva/jwt-go"
)

//go:embed jwt_key
var privateKey []byte

//go:embed jwt_key.pub
var publicKey []byte

var rsaPrivateKey = func() *rsa.PrivateKey {
	rsaPKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		panic(err)
	}
	return rsaPKey
}()

var rsaPublicKey = func() *rsa.PublicKey {
	rsaPKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		panic(err)
	}
	return rsaPKey
}()
