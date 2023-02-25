package jwt

import (
	"fmt"
	"testing"
	"time"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/dgrijalva/jwt-go"
)

func TestJWT(t *testing.T) {
	//key, err := fPrivate.ReadFile("jwt_key")
	////key, err := ioutil.ReadFile("jwt_key")
	//if err != nil {
	//	t.Fatalf("err %v", err)
	//}
	//key := privateKey
	//rsaPKey, err := jwt.ParseRSAPrivateKeyFromPEM(key)
	//if err != nil {
	//	t.Fatalf("err111 %v", err)
	//}
	//rsaPKey := rsaPrivateKey

	//claims := jwt.Claims()
	//uClaims := &UserClaims{
	//	UserName: "222",
	//	StandardClaims: jwt.StandardClaims{
	//		ExpiresAt: time.Now().Add(time.Duration(100) * time.Second).Unix(),
	//	},
	//}
	//
	//token := jwt.NewWithClaims(jwt.SigningMethodRS256, uClaims)
	//s, e := token.SignedString(rsaPKey)
	//if e != nil {
	//	t.Fatalf("e  %v", e)
	//}

	s, e := GetUserToken("22233", time.Second*1000)
	nlog.Info("token %v\n%v", e, s)
	{
		u, e := ParseUserToken(s)
		nlog.Info("e %v u %+v", e, u)
	}
}

func TestJWTParseToken(t *testing.T) {
	//key, err := fPublic.ReadFile("jwt_key.pub")
	////key, err := ioutil.ReadFile("jwt_key.pub")
	//if err != nil {
	//	t.Fatalf("err %v", err)
	//}
	key := publicKey

	tokenStr := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyTmFtZSI6IjIyMiIsIlBXRCI6IjMzMyIsImV4cCI6MTY1NDA4MDIwMX0.KbLdoGzcD0rnryBlmpWr0kWVJi_HJP_vlYf-my1536mEPdASlwDyYp81oqWtHBUnr7SH1iQ9uDLLZNjdAOlOdotEoOwR-EJ1kTEgJF121MPwOGRDWGm0dUW9O2tbIDDoTt8Hk0CsUGkVshN88EJo-j8nkcLfcOPh02zj2nxjtUp9iHEYwaQjjvEuTfB9Gc6j_FD04mR0QBzDpJas0msQaPvWHJzCP4PhB3HhbSxrXDuOhgyzwg-LiQ0Sv3JLosb24-F5dtW6vK3bfBIYb0KglgTMCY-dRva6b6UdYk4rPLBNdI267bXK-NQyxo7tLIsWR2hMJJsuFJTjobGLPD4prA`
	rsaPKey, err := jwt.ParseRSAPublicKeyFromPEM(key)
	if err != nil {
		t.Fatalf("err111 %v", err)
	}

	token, err := jwt.ParseWithClaims(tokenStr, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexept singing methods %v", token.Header["alg"])
		}
		return rsaPKey, nil
	})

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		nlog.Info("ok %v  %v", ok, token.Valid)
		return
	}

	nlog.Info("ok %v  %v", ok, token.Valid)
	nlog.Info("claims %v", claims.UserName)
}

func TestGetRoleToken(t *testing.T) {
	// token := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyTmFtZSI6Imh4ZDExIiwiU2VsZWN0Um9sZUlEIjoxNTMyMjYwMTc1NjAyMjk0Nzg0LCJleHAiOjE2NTQ3NjAzODd9.NkyhPoTPlYqePJCc1yQTIFJnNRP0i2D1DeHwNKNqNA0EftY2nPOSTlpaDsYo9B22OCzm-ipgCIVK_fg_b-DnCcMpbo8OGMpJPXDqDgYHKh6qddqKFc0TZYxg50Jd9txMpAfdvmvZmJcfoYm3i7gzePH_N4xZT7uLKBEu5ffba4dZxKLv9gE8q9RaSiJumcRoOSBc_u9f1Iv-qBvoK6IyMzI37T6cBe8tjm6RhTY0ezDmxvyrsYO7z2i9ddO69tbcNiXpKr519fJiqGQdsKpPa9sYJFyKXolIqM-p7tuJ2PHjwSIEmAMHuAMWkgIJ2AKl_cEWX8l4KsL7lYjiqeoubw`
	token := `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJVc2VyTmFtZSI6Imh4ZDExIiwiU2VsZWN0Um9sZUlEIjoxNTMyMjYwMTc1NjAyMjk0Nzg0LCJHU0lEIjozLCJleHAiOjE2NTQ3NjA1NTR9.pPdBZl_KKKG65oWkMUVZTUPAQJuvpwToB2eJfZk9sXoXXDZtHpzxxR9ZROeQfAcEZScOvcEsdJyEQauJE-tDEzCxkLlpzbYgGHcSmnWRGGM3XPOtn5KMrw9p7k9siqo8mTwFl-uBWRVLwEXMxNCXyY4VKgZ-CJNICuPYnBqd5t5fkT_lrTAFyhptyhFwl8xyNe-KW9JbGklMOcHUX_SihaCqZgkEnWzxHwVO8_6l0bQu_mHdV0niyAHfnpjIdjHy10SD0QYZ5qwHThli1rB1n28s0qz5enVKsyqpYWW3c5zXb-XgqvfLQkREtHfbKmg7v3QucnAdMh0jlNCfLG0uSQ`
	role, err := ParseRoleToken(token)

	nlog.Info("err %v %+v", err, role)
}
