package builder

import (
	"context"
	"encoding/json"

	"github.com/rancher/kim/pkg/client"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubernetes/pkg/credentialprovider"
)

type Login struct {
	Password      string `usage:"Password" short:"p"`
	PasswordStdin bool   `usage:"Take the password from stdin"`
	Username      string `usage:"Username" short:"u"`
}

func (s *Login) Do(_ context.Context, k *client.Interface, server string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		login, err := k.Core.Secret().Get(k.Namespace, "kim-docker-config", metav1.GetOptions{})
		if apierr.IsNotFound(err) {
			dockerConfigJSON := credentialprovider.DockerConfigJSON{
				Auths: map[string]credentialprovider.DockerConfigEntry{
					server: {
						Username: s.Username,
						Password: s.Password,
					},
				},
			}
			dockerConfigJSONBytes, err := json.Marshal(&dockerConfigJSON)
			if err != nil {
				return err
			}
			login = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kim-docker-config",
					Namespace: k.Namespace,
					Labels: labels.Set{
						"app.kubernetes.io/managed-by": "kim",
					},
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: dockerConfigJSONBytes,
				},
			}
			_, err = k.Core.Secret().Create(login)
			return err
		}
		var dockerConfigJSON credentialprovider.DockerConfigJSON
		if dockerConfigJSONBytes, ok := login.Data[corev1.DockerConfigJsonKey]; ok {
			if err := json.Unmarshal(dockerConfigJSONBytes, &dockerConfigJSON); err != nil {
				return err
			}
		}
		dockerConfigJSON.Auths[server] = credentialprovider.DockerConfigEntry{
			Username: s.Username,
			Password: s.Password,
		}
		dockerConfigJSONBytes, err := json.Marshal(&dockerConfigJSON)
		if err != nil {
			return err
		}
		login.Type = corev1.SecretTypeDockerConfigJson
		login.Data[corev1.DockerConfigJsonKey] = dockerConfigJSONBytes
		_, err = k.Core.Secret().Update(login)
		return err
	})
}
