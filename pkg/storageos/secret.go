package storageos

import (
	corev1 "k8s.io/api/core/v1"
)

// TLSEtcdSecretName is the name of secret resource that contains etcd TLS
// secrets.
const TLSEtcdSecretName = "storageos-tls-etcd"

func (s *Deployment) createInitSecret() error {
	username, password, err := s.getAdminCreds()
	if err != nil {
		return err
	}
	data := map[string][]byte{
		credUsernameKey: username,
		credPasswordKey: password,
	}
	return s.k8sResourceManager.Secret(initSecretName, s.stos.Spec.GetResourceNS(), corev1.SecretTypeOpaque, data).Create()
}

func (s *Deployment) createTLSSecret() error {
	cert, key, err := s.getTLSData()
	if err != nil {
		return err
	}
	data := map[string][]byte{
		tlsCertKey: cert,
		tlsKeyKey:  key,
	}
	return s.k8sResourceManager.Secret(tlsSecretName, s.stos.Spec.GetResourceNS(), corev1.SecretTypeTLS, data).Create()
}

func (s *Deployment) getAdminCreds() ([]byte, []byte, error) {
	var username, password []byte
	if s.stos.Spec.SecretRefName != "" && s.stos.Spec.SecretRefNamespace != "" {
		secret, err := s.k8sResourceManager.Secret(s.stos.Spec.SecretRefName, s.stos.Spec.SecretRefNamespace, corev1.SecretTypeOpaque, nil).Get()
		if err != nil {
			return nil, nil, err
		}
		data := secret.Data

		username = data[apiUsernameKey]
		password = data[apiPasswordKey]
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
		secret, err := s.k8sResourceManager.Secret(s.stos.Spec.SecretRefName, s.stos.Spec.SecretRefNamespace, corev1.SecretTypeTLS, nil).Get()
		if err != nil {
			return nil, nil, err
		}
		data := secret.Data

		cert = data[tlsCertKey]
		key = data[tlsKeyKey]
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
		data := map[string][]byte{
			credUsernameKey: username,
			credPasswordKey: password,
		}
		if err := s.k8sResourceManager.Secret(csiProvisionerSecretName, s.stos.Spec.GetResourceNS(), corev1.SecretTypeOpaque, data).Create(); err != nil {
			return err
		}
	}

	// Create Controller Publish Secret.
	if s.stos.Spec.CSI.EnableControllerPublishCreds {
		username, password, err := s.getCSICreds(csiControllerPublishUsernameKey, csiControllerPublishPasswordKey)
		if err != nil {
			return err
		}
		data := map[string][]byte{
			credUsernameKey: username,
			credPasswordKey: password,
		}
		if err := s.k8sResourceManager.Secret(csiControllerPublishSecretName, s.stos.Spec.GetResourceNS(), corev1.SecretTypeOpaque, data).Create(); err != nil {
			return err
		}
	}

	// Create Node Publish Secret.
	if s.stos.Spec.CSI.EnableNodePublishCreds {
		username, password, err := s.getCSICreds(csiNodePublishUsernameKey, csiNodePublishPasswordKey)
		if err != nil {
			return err
		}
		data := map[string][]byte{
			credUsernameKey: username,
			credPasswordKey: password,
		}
		if err := s.k8sResourceManager.Secret(csiNodePublishSecretName, s.stos.Spec.GetResourceNS(), corev1.SecretTypeOpaque, data).Create(); err != nil {
			return err
		}
	}

	return nil
}

// deleteCSISecrets deletes all the CSI related secrets.
func (s *Deployment) deleteCSISecrets() error {
	namespace := s.stos.Spec.GetResourceNS()
	if err := s.k8sResourceManager.Secret(csiProvisionerSecretName, namespace, corev1.SecretTypeOpaque, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.Secret(csiControllerPublishSecretName, namespace, corev1.SecretTypeOpaque, nil).Delete(); err != nil {
		return err
	}

	return nil
}

// getCSICreds - given username and password keys, it fetches the creds from
// storageos-api secret and returns them.
func (s *Deployment) getCSICreds(usernameKey, passwordKey string) (username []byte, password []byte, err error) {
	// Get the username and password from storageos-api secret object.
	secret, err := s.k8sResourceManager.Secret(s.stos.Spec.SecretRefName, s.stos.Spec.SecretRefNamespace, corev1.SecretTypeOpaque, nil).Get()
	if err != nil {
		return nil, nil, err
	}
	data := secret.Data

	username = data[usernameKey]
	password = data[passwordKey]

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
	secret, err := s.k8sResourceManager.Secret(s.stos.Spec.TLSEtcdSecretRefName, s.stos.Spec.TLSEtcdSecretRefNamespace, corev1.SecretTypeOpaque, nil).Get()
	if err != nil {
		return err
	}
	data := secret.Data

	// Create new secret with etcd TLS secret data.
	return s.k8sResourceManager.Secret(TLSEtcdSecretName, s.stos.Spec.GetResourceNS(), corev1.SecretTypeTLS, data).Create()
}
