// +build !linux

package action

import (
	"context"
	"runtime"

	"github.com/pkg/errors"
)

func (_ *Agent) Run(_ context.Context) error {
	return errors.Errorf("patform not supported: %s/%s", runtime.GOOS, runtime.GOARCH)
}
