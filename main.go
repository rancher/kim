//go:generate protoc --gofast_out=plugins=grpc:. -I=./vendor:. pkg/apis/services/images/v1alpha1/images.proto

package main

import (
	"os"
	"path/filepath"

	"github.com/containerd/containerd/pkg/seed"
	"github.com/rancher/kim/pkg/cli"
	command "github.com/rancher/wrangler-cli"

	// Add non-default auth providers
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func init() {
	seed.WithTimeAndRand()
}

func main() {
	switch _, exe := filepath.Split(os.Args[0]); exe {
	case "kubectl-builder":
		command.Main(cli.Builder(exe))
	case "kubectl-image":
		command.Main(cli.Image(exe))
	default:
		command.Main(cli.Main())
	}
}
