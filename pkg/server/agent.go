package server

type Agent struct {
	Config
	Tlscacert string `usage:"ca certificate to verify clients"`
	Tlscert   string `usage:"server tls certificate"`
	Tlskey    string `usage:"server tls key"`
}
