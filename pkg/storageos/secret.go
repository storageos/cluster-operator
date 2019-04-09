package storageos

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TLSEtcdSecretName is the name of secret resource that contains etcd TLS
// secrets.
const TLSEtcdSecretName = "storageos-tls-etcd"

func (s *Deployment) deleteSecret(name string) error {
	return s.deleteObject(s.getSecret(name))
}

func (s *Deployment) getSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

func (s *Deployment) createInitSecret() error {
	username, password, err := s.getAdminCreds()
	if err != nil {
		return err
	}
	if err := s.createCredSecret(initSecretName, username, password); err != nil {
		return err
	}
	return nil
}

func (s *Deployment) createTLSSecret() error {
	cert, key, err := s.getTLSData()
	if err != nil {
		return err
	}

	secret := s.getSecret(tlsSecretName)
	secret.Type = corev1.SecretType(tlsSecretType)
	secret.Data = map[string][]byte{
		tlsCertKey: cert,
		tlsKeyKey:  key,
	}
	return s.createOrUpdateObject(secret)
}

func (s *Deployment) getAdminCreds() ([]byte, []byte, error) {
	var username, password []byte
	if s.stos.Spec.SecretRefName != "" && s.stos.Spec.SecretRefNamespace != "" {
		se := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.stos.Spec.SecretRefName,
				Namespace: s.stos.Spec.SecretRefNamespace,
			},
		}
		nsName := types.NamespacedName{
			Name:      se.ObjectMeta.GetName(),
			Namespace: se.ObjectMeta.GetNamespace(),
		}
		if err := s.client.Get(context.Background(), nsName, se); err != nil {
			return nil, nil, err
		}

		username = se.Data[apiUsernameKey]
		password = se.Data[apiPasswordKey]
	} else {
		// Use the default credentials.
		username = []byte(defaultUsername)
		password = []byte(defaultPassword)
	}

	return username, password, nil
}

func (s *Deployment) getTLSData() ([]byte, []byte, error) {
	var cert, key []byte
	if s.stos.Spec.SecretRefName != "" && s.stos.Spec.SecretRefNamespace != "" {
		se := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.stos.Spec.SecretRefName,
				Namespace: s.stos.Spec.SecretRefNamespace,
			},
		}
		nsName := types.NamespacedName{
			Name:      se.ObjectMeta.GetName(),
			Namespace: se.ObjectMeta.GetNamespace(),
		}
		if err := s.client.Get(context.Background(), nsName, se); err != nil {
			return nil, nil, err
		}

		cert = se.Data[tlsCertKey]
		key = se.Data[tlsKeyKey]
	} else {
		cert = []byte("")
		key = []byte("")
	}

	return cert, key, nil
}

// createCSISecrets checks which CSI creds are enabled and creates secret for
// those components.
func (s *Deployment) createCSISecrets() error {
	// Create Provision Secret.
	if s.stos.Spec.CSI.EnableProvisionCreds {
		username, password, err := s.getCSICreds(csiProvisionUsernameKey, csiProvisionPasswordKey)
		if err != nil {
			return err
		}
		if err := s.createCredSecret(csiProvisionerSecretName, username, password); err != nil {
			return err
		}
	}

	// Create Controller Publish Secret.
	if s.stos.Spec.CSI.EnableControllerPublishCreds {
		username, password, err := s.getCSICreds(csiControllerPublishUsernameKey, csiControllerPublishPasswordKey)
		if err != nil {
			return err
		}
		if err := s.createCredSecret(csiControllerPublishSecretName, username, password); err != nil {
			return err
		}
	}

	// Create Node Publish Secret.
	if s.stos.Spec.CSI.EnableNodePublishCreds {
		username, password, err := s.getCSICreds(csiNodePublishUsernameKey, csiNodePublishPasswordKey)
		if err != nil {
			return err
		}
		if err := s.createCredSecret(csiNodePublishSecretName, username, password); err != nil {
			return err
		}
	}

	return nil
}

// deleteCSISecrets deletes all the CSI related secrets.
func (s *Deployment) deleteCSISecrets() error {
	if err := s.deleteSecret(csiProvisionerSecretName); err != nil {
		return err
	}

	if err := s.deleteSecret(csiControllerPublishSecretName); err != nil {
		return err
	}

	if err := s.deleteSecret(csiNodePublishSecretName); err != nil {
		return err
	}

	return nil
}

// createCredSecret creates a credential type secret with username and password.
func (s *Deployment) createCredSecret(name string, username, password []byte) error {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
		Type: corev1.SecretType(corev1.SecretTypeOpaque),
		Data: map[string][]byte{
			"username": username,
			"password": password,
		},
	}

	return s.createOrUpdateObject(secret)
}

// getCSICreds - given username and password keys, it fetches the creds from
// storageos-api secret and returns them.
func (s *Deployment) getCSICreds(usernameKey, passwordKey string) (username []byte, password []byte, err error) {
	// Get the username and password from storageos-api secret object.
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.stos.Spec.SecretRefName,
			Namespace: s.stos.Spec.SecretRefNamespace,
		},
	}
	nsName := types.NamespacedName{
		Name:      secret.ObjectMeta.GetName(),
		Namespace: secret.ObjectMeta.GetNamespace(),
	}
	if err := s.client.Get(context.Background(), nsName, secret); err != nil {
		return nil, nil, err
	}

	username = secret.Data[usernameKey]
	password = secret.Data[passwordKey]

	return username, password, err
}

// createTLSEtcdSecret creates a new TLS secret in the deployment namespace by
// copying the secret data from the etcd TLS secret reference so that it can be
// referred by other resources in the deployment namespace.
func (s *Deployment) createTLSEtcdSecret() error {
	if s.stos.Spec.TLSEtcdSecretRefName == "" &&
		s.stos.Spec.TLSEtcdSecretRefNamespace == "" {
		// No etcd TLS secret reference specified.
		return nil
	}

	// Fetch etcd TLS secret.
	existingSecret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Name:      s.stos.Spec.TLSEtcdSecretRefName,
		Namespace: s.stos.Spec.TLSEtcdSecretRefNamespace,
	}
	if err := s.client.Get(context.Background(), nsName, existingSecret); err != nil {
		return err
	}

	// Create new secret with etcd TLS secret data.
	secret := s.getSecret(TLSEtcdSecretName)
	secret.Type = existingSecret.Type
	secret.Data = existingSecret.Data

	return s.createOrUpdateObject(secret)
}
