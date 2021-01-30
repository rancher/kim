package action

import "github.com/rancher/kim/pkg/server"

type Agent struct {
	server.Config
	Tlscacert string `usage:"ca certificate to verify clients"`
	Tlscert   string `usage:"server tls certificate"`
	Tlskey    string `usage:"server tls key"`
}
