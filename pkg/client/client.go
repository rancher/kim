package client

import (
	"context"
	"net"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/apply"
	appsctl "github.com/rancher/wrangler/pkg/generated/controllers/apps"
	appsctlv1 "github.com/rancher/wrangler/pkg/generated/controllers/apps/v1"
	corectl "github.com/rancher/wrangler/pkg/generated/controllers/core"
	corectlv1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	rbacctl "github.com/rancher/wrangler/pkg/generated/controllers/rbac"
	rbacctlv1 "github.com/rancher/wrangler/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultNamespace = "kube-image"
)

var DefaultConfig = Config{
	Namespace: DefaultNamespace,
}

type Config struct {
	Namespace  string `usage:"namespace" short:"n" env:"NAMESPACE" default:"kube-image"`
	Kubeconfig string `usage:"kubeconfig for authentication" short:"k" env:"KUBECONFIG"`
	Context    string `usage:"kubeconfig context for authentication" short:"x" env:"KUBECONTEXT"`
}

func (c *Config) Interface() (*Interface, error) {
	if c == nil {
		return nil, errors.Errorf("client is not configured, please set client config")
	}
	return NewInterface(c.Kubeconfig, c.Context, c.Namespace)
}

type Interface struct {
	Core      corectlv1.Interface
	Apps      appsctlv1.Interface
	RBAC      rbacctlv1.Interface
	Apply     apply.Apply
	Namespace string
}

func NewInterface(kubecfg, kubectx, kubens string) (*Interface, error) {
	cc := kubeconfig.GetNonInteractiveClientConfigWithContext(kubecfg, kubectx)
	ns, _, err := cc.Namespace()
	if err != nil {
		return nil, err
	}

	if kubens != "" {
		ns = kubens
	}

	rc, err := cc.ClientConfig()
	if err != nil {
		return nil, err
	}

	c := &Interface{
		Namespace: ns,
	}

	core, err := corectl.NewFactoryFromConfig(rc)
	if err != nil {
		return nil, err
	}
	c.Core = core.Core().V1()

	apps, err := appsctl.NewFactoryFromConfig(rc)
	if err != nil {
		return nil, err
	}
	c.Apps = apps.Apps().V1()

	rbac, err := rbacctl.NewFactoryFromConfig(rc)
	if err != nil {
		return nil, err
	}
	c.RBAC = rbac.Rbac().V1()

	c.Apply, err = apply.NewForConfig(rc)
	if err != nil {
		return nil, err
	}

	if c.Namespace == "" {
		c.Namespace = DefaultNamespace
	}

	c.Apply = c.Apply.
		WithDynamicLookup().
		WithDefaultNamespace(c.Namespace).
		WithListerNamespace(c.Namespace).
		WithRestrictClusterScoped()

	return c, nil
}

func GetServiceAddress(_ context.Context, k8s *Interface, port string) (string, error) {
	// TODO handle multiple addresses
	service, err := k8s.Core.Service().Get(k8s.Namespace, "builder", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	endpoints, err := k8s.Core.Endpoints().Get(k8s.Namespace, "builder", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for _, sub := range endpoints.Subsets {
		if len(sub.Addresses) > 0 {
			for _, p := range sub.Ports {
				if p.Name == port {
					host := sub.Addresses[0].IP
					if override, ok := service.Annotations["images.cattle.io/endpoint-override"]; ok {
						host = override
					}
					return net.JoinHostPort(host, strconv.FormatInt(int64(p.Port), 10)), nil
				}
			}
		}
	}
	return "", errors.New("unknown service port")
}
