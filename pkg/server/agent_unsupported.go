// +build !linux

package server

import (
	"context"
	"runtime"

	"github.com/pkg/errors"
)

func (a *Agent) Run(_ context.Context) error {
	return errors.Errorf("patform not supported: %s/%s", runtime.GOOS, runtime.GOARCH)
}
