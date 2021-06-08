package builder

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/kim/pkg/client"
	"github.com/rancher/kim/pkg/server"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
)

// Install the builder.
type Install struct {
	Force    bool   `usage:"Force installation by deleting existing builder"`
	Selector string `usage:"Selector for nodes (label query) to apply builder role"`
	NoWait   bool   `usage:"Do not wait for backend to become available"`
	NoFail   bool   `usage:"Do not fail if backend components are already installed"`
	server.Config
}

func (a *Install) checkNoFail(err error) error {
	if err == nil {
		return nil
	}
	if a.NoFail {
		logrus.Warn(err)
		return nil
	}
	return err
}

func (a *Install) Do(ctx context.Context, k8s *client.Interface) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// assert node-role
	if err := a.NodeRole(ctx, k8s); err != nil {
		return a.checkNoFail(err)
	}
	// assert namespace
	if err := a.Namespace(ctx, k8s); err != nil {
		return a.checkNoFail(err)
	}
	// assert secrets
	if err := a.Secrets(ctx, k8s); err != nil {
		return a.checkNoFail(err)
	}
	// assert service
	if err := a.Service(ctx, k8s); err != nil {
		return a.checkNoFail(err)
	}
	// assert daemonset
	if err := a.DaemonSet(ctx, k8s); err != nil {
		return a.checkNoFail(err)
	}

	if a.NoWait {
		return nil
	}

	retryMe := errors.New("timeout waiting for builder to become available")
	return retry.OnError(
		wait.Backoff{
			Steps:    15,
			Duration: 5 * time.Second,
			Factor:   1.0,
			Jitter:   0.3,
		},
		func(err error) bool {
			return err == retryMe
		},
		func() error {
			daemon, err := k8s.Apps.DaemonSet().Get(k8s.Namespace, "builder", metav1.GetOptions{})
			if err != nil {
				return err
			}
			if daemon.Status.NumberReady == 0 {
				logrus.Infof("Waiting on builder daemon availability...")
				return retryMe
			}
			return nil
		},
	)
}

func (_ *Install) Namespace(_ context.Context, k *client.Interface) error {
	logrus.Infof("Asserting namespace `%s`", k.Namespace)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		ns, err := k.Core.Namespace().Get(k.Namespace, metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: k.Namespace,
					Labels: labels.Set{
						"app.kubernetes.io/managed-by": "kim",
					},
				},
			}
			ns, err = k.Core.Namespace().Create(ns)
			return err
		}
		if ns.Labels == nil {
			ns.Labels = labels.Set{}
		}
		if _, ok := ns.Labels["app.kubernetes.io/managed-by"]; !ok {
			ns.Labels["app.kubernetes.io/managed-by"] = "kim"
		}
		ns, err = k.Core.Namespace().Update(ns)
		return err
	})
}

func (a *Install) Secrets(_ context.Context, k *client.Interface) error {
	logrus.Info("Asserting TLS secrets")
	secrets := k.Core.Secret()
	if a.Force {
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions := metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
		secrets.Delete(k.Namespace, "kim-tls-client", &deleteOptions)
		secrets.Delete(k.Namespace, "kim-tls-server", &deleteOptions)
		secrets.Delete(k.Namespace, "kim-tls-ca", &deleteOptions)
	}

	// assert CA
	caCert, caKey, err := client.LoadOrGenCA(secrets, k.Namespace, "kim-tls-ca")
	if err != nil {
		return errors.Wrap(err, "failed to assert certificate authority")
	}
	nodeList, err := k.Core.Node().List(metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/builder==true",
	})
	if err != nil {
		return err
	}

	ips := []net.IP{}
	domains := []string{
		fmt.Sprintf("builder.%s.svc", k.Namespace),
	}
	for _, node := range nodeList.Items {
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				ips = append(ips, net.ParseIP(addr.Address))
			case corev1.NodeHostName:
				domains = append(domains, addr.Address)
			}
		}
	}
	// assert server cert+key
	_, _, err = client.LoadOrGenServerCert(secrets, k.Namespace, "kim-tls-server", caCert, caKey, "kube-image-server", nil, domains, ips)
	if err != nil {
		return errors.Wrap(err, "failed to assert server cert+key")
	}
	// assert client cert+key
	_, _, err = client.LoadOrGenClientCert(secrets, k.Namespace, "kim-tls-client", caCert, caKey, "kube-image-client")
	if err != nil {
		return errors.Wrap(err, "failed to assert client cert+key")
	}
	return nil
}

func (a *Install) Service(_ context.Context, k *client.Interface) error {
	logrus.Info("Asserting service/endpoints")
	if a.Force {
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions := metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
		k.Core.Service().Delete(k.Namespace, "builder", &deleteOptions)
	}
	if a.AgentPort <= 0 {
		a.AgentPort = server.DefaultAgentPort
	}
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		svc, err := k.Core.Service().Get(k.Namespace, "builder", metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			svc = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "builder",
					Namespace: k.Namespace,
					Labels: labels.Set{
						"app.kubernetes.io/managed-by": "kim",
					},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Selector: labels.Set{
						"app.kubernetes.io/name":      "kim",
						"app.kubernetes.io/component": "builder",
					},
					Ports: []corev1.ServicePort{
						a.servicePort("buildkit"),
						a.servicePort("kim"),
					},
				},
			}
			svc, err = k.Core.Service().Create(svc)
			return err
		}
		if svc.Labels == nil {
			svc.Labels = labels.Set{}
		}
		if _, ok := svc.Labels["app.kubernetes.io/managed-by"]; !ok {
			svc.Labels["app.kubernetes.io/managed-by"] = "kim"
		}
		svc, err = k.Core.Service().Update(svc)
		return err
	})
}

func (a *Install) DaemonSet(_ context.Context, k *client.Interface) error {
	logrus.Info("Installing builder daemon")
	if a.Force {
		deletePropagation := metav1.DeletePropagationBackground
		deleteOptions := metav1.DeleteOptions{
			PropagationPolicy: &deletePropagation,
		}
		k.Apps.DaemonSet().Delete(k.Namespace, "builder", &deleteOptions)
	}

	agentImage, err := a.GetAgentImage()
	if err != nil {
		return err
	}

	buildkitImage, err := a.GetBuildkitImage()
	if err != nil {
		return err
	}
	buildkitProbe := corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{"buildctl", "debug", "workers"},
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       20,
	}

	privileged := true
	hostPathDirectory := corev1.HostPathDirectory
	hostPathDirectoryOrCreate := corev1.HostPathDirectoryOrCreate
	mountPropagationBidirectional := corev1.MountPropagationBidirectional

	daemon := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "builder",
			Namespace: k.Namespace,
			Labels: labels.Set{
				"app.kubernetes.io/name":       "kim",
				"app.kubernetes.io/component":  "builder",
				"app.kubernetes.io/managed-by": "kim",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.Set{
					"app":       "kim",
					"component": "builder",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels.Set{
						"app":                          "kim",
						"component":                    "builder",
						"app.kubernetes.io/name":       "kim",
						"app.kubernetes.io/component":  "builder",
						"app.kubernetes.io/managed-by": "kim",
					},
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					HostPID:     true,
					HostIPC:     true,
					NodeSelector: labels.Set{
						"node-role.kubernetes.io/builder": "true",
					},
					DNSPolicy: corev1.DNSClusterFirstWithHostNet,
					InitContainers: []corev1.Container{{
						Name:  "rshared-tmp",
						Image: buildkitImage,
						Env: []corev1.EnvVar{
							{Name: "_DIR", Value: "/tmp"},
							{Name: "_PATH", Value: "/usr/sbin:/usr/bin:/sbin:/bin:/bin/aux"},
						},
						Command: []string{"sh", "-c"},
						Args:    []string{"(if mountpoint $_DIR; then set -x; nsenter -m -p -t 1 -- env PATH=$_PATH sh -c 'mount --make-rshared $_DIR'; fi) || true"},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "host-tmp", MountPath: "/tmp"},
						},
					}, {
						Name:  "rshared-buildkit",
						Image: buildkitImage,
						Env: []corev1.EnvVar{
							{Name: "_DIR", Value: "/var/lib/buildkit"},
							{Name: "_PATH", Value: "/usr/sbin:/usr/bin:/sbin:/bin:/bin/aux"},
						},
						Command: []string{"sh", "-c"},
						Args:    []string{"(if mountpoint $_DIR; then set -x; nsenter -m -p -t 1 -- env PATH=$_PATH sh -c 'mount --make-rshared $_DIR'; fi) || true"},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "host-var-lib-buildkit", MountPath: "/var/lib/buildkit"},
						},
					}, {
						Name:  "rshared-containerd",
						Image: buildkitImage,
						Env: []corev1.EnvVar{
							{Name: "_DIR", Value: a.ContainerdVolume},
							{Name: "_PATH", Value: "/usr/sbin:/usr/bin:/sbin:/bin:/bin/aux"},
						},
						Command: []string{"sh", "-c"},
						Args:    []string{"(if mountpoint $_DIR; then set -x; nsenter -m -p -t 1 -- env PATH=$_PATH sh -c 'mount --make-rshared $_DIR'; fi) || true"},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "host-containerd", MountPath: a.ContainerdVolume},
						},
					}},
					Containers: []corev1.Container{{
						Name:  "buildkit",
						Image: buildkitImage,
						Args: []string{
							fmt.Sprintf("--addr=%s", a.BuildkitSocket),
							fmt.Sprintf("--addr=tcp://0.0.0.0:%d", a.BuildkitPort),
							"--containerd-worker=true",
							fmt.Sprintf("--containerd-worker-addr=%s", a.ContainerdSocket),
							"--containerd-worker-gc",
							"--oci-worker=false",
							"--tlscacert=/certs/ca/tls.crt",
							"--tlscert=/certs/server/tls.crt",
							"--tlskey=/certs/server/tls.key",
						},
						Ports: []corev1.ContainerPort{
							a.containerPort("buildkit"),
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "host-ctl", MountPath: "/sys/fs/cgroup"},
							{Name: "host-run", MountPath: "/run"},
							{Name: "host-tmp", MountPath: "/tmp", MountPropagation: &mountPropagationBidirectional},
							{Name: "host-var-lib-buildkit", MountPath: "/var/lib/buildkit", MountPropagation: &mountPropagationBidirectional},
							{Name: "host-containerd", MountPath: a.ContainerdVolume, MountPropagation: &mountPropagationBidirectional},
							{Name: "certs-ca", MountPath: "/certs/ca", ReadOnly: true},
							{Name: "certs-server", MountPath: "/certs/server", ReadOnly: true},
						},
						ReadinessProbe: &buildkitProbe,
						LivenessProbe:  &buildkitProbe,
					}, {
						Name:    "agent",
						Image:   agentImage,
						Command: []string{"kim", "--debug", "agent"},
						Args: []string{
							fmt.Sprintf("--agent-port=%d", a.AgentPort),
							fmt.Sprintf("--buildkit-socket=%s", a.BuildkitSocket),
							fmt.Sprintf("--buildkit-port=%d", a.BuildkitPort),
							fmt.Sprintf("--containerd-socket=%s", a.ContainerdSocket),
							"--tlscacert=/certs/ca/tls.crt",
							"--tlscert=/certs/server/tls.crt",
							"--tlskey=/certs/server/tls.key",
						},
						Ports: []corev1.ContainerPort{
							a.containerPort("kim"),
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "host-containerd", MountPath: a.ContainerdVolume, MountPropagation: &mountPropagationBidirectional},
							{Name: "host-ctl", MountPath: "/sys/fs/cgroup"},
							{Name: "host-etc-pki", MountPath: "/etc/pki", ReadOnly: true},
							{Name: "host-etc-ssl", MountPath: "/etc/ssl", ReadOnly: true},
							{Name: "host-run", MountPath: "/run"},
							{Name: "host-var-lib-buildkit", MountPath: "/var/lib/buildkit", MountPropagation: &mountPropagationBidirectional},
							{Name: "certs-ca", MountPath: "/certs/ca", ReadOnly: true},
							{Name: "certs-server", MountPath: "/certs/server", ReadOnly: true},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "host-ctl", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys/fs/cgroup", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "host-etc-pki", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/pki", Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "host-etc-ssl", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/ssl", Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "host-run", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "host-tmp", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/tmp", Type: &hostPathDirectory,
								},
							},
						},
						{
							Name: "host-var-lib-buildkit", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/buildkit", Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "host-containerd", VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: a.ContainerdVolume, Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "certs-ca", VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "kim-tls-ca",
								},
							},
						},
						{
							Name: "certs-server", VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "kim-tls-server",
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = k.Apps.DaemonSet().Create(daemon)
	if apierr.IsAlreadyExists(err) {
		return errors.Errorf("builder already installed")
	}
	return err
}

// NodeRole asserts that the node can run KIM and labels it with the builder role
func (a *Install) NodeRole(_ context.Context, k *client.Interface) error {
	nodeList, err := k.Core.Node().List(metav1.ListOptions{
		LabelSelector: a.Selector,
	})
	if err != nil {
		return err
	}
	if len(nodeList.Items) == 0 {
		return errors.New("failed to select any nodes")
	}
	if len(nodeList.Items) == 1 {
		logrus.Infof("Applying node-role `builder` to `%s`", nodeList.Items[0].Name)
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			node, err := k.Core.Node().Get(nodeList.Items[0].Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			// detect container runtime and adjust defaults
			crv, err := url.Parse(node.Status.NodeInfo.ContainerRuntimeVersion)
			if err != nil {
				return errors.Wrap(err, "failed to parse container runtime version")
			}
			switch {
			// embedded containerd
			case crv.Scheme == "containerd" && strings.Contains(crv.Host, "-k3s"):
				if a.ContainerdSocket == "" {
					a.ContainerdSocket = server.K3sContainerdSocket
				}
				if a.ContainerdVolume == "" {
					a.ContainerdVolume = server.K3sContainerdVolume
				}
			// external containerd
			case crv.Scheme == "containerd" /* && !strings.Contains(crv.Host, "-k3s") */ :
				if a.ContainerdSocket == "" {
					a.ContainerdSocket = server.StockContainerdSocket
				}
				if a.ContainerdVolume == "" {
					a.ContainerdVolume = server.StockContainerdVolume
				}
			default:
				return errors.Errorf("container runtime `%s` not supported", crv.Scheme)
			}
			node.Labels = labels.Merge(node.Labels, labels.Set{
				"node-role.kubernetes.io/builder": "true",
			})
			_, err = k.Core.Node().Update(node)
			return err
		})
	}

	label := "k3s.io/hostname"
	if _, k3s := nodeList.Items[0].Labels[label]; !k3s {
		label = "kubernetes.io/hostname"
	}
	return errors.Errorf("Too many nodes, please specify a selector, e.g. %s=%s", label, nodeList.Items[0].Name)
}

func (a *Install) containerPort(name string) corev1.ContainerPort {
	switch name {
	case "buildkit":
		return corev1.ContainerPort{
			Name:          name,
			ContainerPort: int32(a.BuildkitPort),
			Protocol:      corev1.ProtocolTCP,
		}
	case "kim":
		return corev1.ContainerPort{
			Name:          name,
			ContainerPort: int32(a.AgentPort),
			Protocol:      corev1.ProtocolTCP,
		}
	default:
		return corev1.ContainerPort{Name: name}
	}
}

func (a *Install) servicePort(name string) corev1.ServicePort {
	switch name {
	case "buildkit":
		return corev1.ServicePort{
			Name:     name,
			Port:     int32(a.BuildkitPort),
			Protocol: corev1.ProtocolTCP,
		}
	case "kim":
		return corev1.ServicePort{
			Name:     name,
			Port:     int32(a.AgentPort),
			Protocol: corev1.ProtocolTCP,
		}
	default:
		return corev1.ServicePort{Name: name}
	}
}
