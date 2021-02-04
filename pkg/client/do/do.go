package do

import (
	"context"
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rancher/kim/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetServiceAddress(_ context.Context, k8s *client.Interface, port string) (string, error) {
	// TODO handle multiple addresses
	endpoints, err := k8s.Core.Endpoints().Get(k8s.Namespace, "builder", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, sub := range endpoints.Subsets {
		if len(sub.Addresses) > 0 {
			for _, p := range sub.Ports {
				if p.Name == port {
					return net.JoinHostPort(sub.Addresses[0].IP, strconv.FormatInt(int64(p.Port), 10)), nil
				}
			}
		}
	}
	return "", errors.New("unknown service port")
}
