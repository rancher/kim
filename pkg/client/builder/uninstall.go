package builder

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/kim/pkg/client"
	"github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
)

// Uninstall the builder.
type Uninstall struct {
	Force bool `usage:"Force uninstallation by deleting namespace"`
}

// Namespace uninstalls the builder namespace.
func (a *Uninstall) Namespace(ctx context.Context, k *client.Interface) error {
	ns, err := k.Core.Namespace().Get(k.Namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if !a.Force && (ns.Labels == nil || ns.Labels["app.kubernetes.io/managed-by"] != "kim") {
		return errors.Errorf("namespace not managed by kim")
	}

	deletePropagation := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePropagation,
	}
	// is there a better way to wait for the namespace to actually be deleted?
	done := make(chan struct{})
	informer := k.Core.Namespace().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			close(done)
		},
	})
	go informer.Run(done)
	err = k.Core.Namespace().Delete(k.Namespace, &deleteOptions)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			_, err = k.Core.Namespace().Get(k.Namespace, metav1.GetOptions{})
			if !apierr.IsNotFound(err) {
				continue
			}
			return nil
		}
	}
}

// NodeRole removes the builder role from all nodes with that role.
func (a *Uninstall) NodeRole(_ context.Context, k *client.Interface) error {
	nodeList, err := k.Core.Node().List(metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/builder",
	})
	if err != nil {
		return err
	}
	for _, node := range nodeList.Items {
		if err = retry.RetryOnConflict(retry.DefaultRetry, removeNodeRole(k, node.Name)); err != nil {
			logrus.Warnf("failed to remove builder label from %s", node.Name)
		}
	}
	return nil
}

func removeNodeRole(k *client.Interface, nodeName string) func() error {
	return func() error {
		node, err := k.Core.Node().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if node.Labels == nil {
			return nil
		}
		delete(node.Labels, "node-role.kubernetes.io/builder")
		_, err = k.Core.Node().Update(node)
		return err
	}
}
