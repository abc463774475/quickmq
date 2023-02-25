package jwt

import (
	errors2 "errors"
	"fmt"
	"testing"

	nlog "github.com/abc463774475/my_tool/n_log"
	"github.com/pkg/errors"
)

func TestErrors(t *testing.T) {
	var err error

	err = ErrTokenExpired

	nlog.Info("err %v", err)
	err = errors.Wrap(err, ErrNotTheUserClaims.Error())

	err = errors.Wrap(err, ErrOther.Error())

	nlog.Info("err  %v", err)

	if errors2.Is(err, ErrNotTheUserClaims) {
		nlog.Info("111")
	}

	if errors2.Is(err, ErrTokenExpired) {
		nlog.Info("222")
	}

	if errors2.Is(err, ErrOther) {
		nlog.Info("333")
	}
}

func TestErrorfmt(t *testing.T) {
	err := ErrNotTheUserClaims
	err = fmt.Errorf("token : %w", err)

	if errors2.Is(err, ErrOther) {
		nlog.Info("111111111111")
	}

	nlog.Info(" %v", err)
}
