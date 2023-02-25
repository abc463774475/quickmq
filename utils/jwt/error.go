package jwt

import "errors"

var ErrNotTheUserClaims = errors.New("not the userClaims")

var ErrTokenExpired = errors.New("token has expired")

var ErrOther = errors.New("other")
