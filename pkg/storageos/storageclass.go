package storageos

func (s *Deployment) createStorageClass() error {
	parameters := map[string]string{}

	// Add CSI creds secrets in parameters.
	parameters[csiFSType] = defaultFSType
	if s.stos.Spec.CSI.EnableProvisionCreds {
		parameters[csiProvisionerSecretNameKey] = csiProvisionerSecretName
		parameters[csiProvisionerSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
	}
	if s.stos.Spec.CSI.EnableControllerPublishCreds {
		parameters[csiControllerPublishSecretNameKey] = csiControllerPublishSecretName
		parameters[csiControllerPublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
	}
	if s.stos.Spec.CSI.EnableNodePublishCreds {
		parameters[csiNodePublishSecretNameKey] = csiNodePublishSecretName
		parameters[csiNodePublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
	}
	// Add expand parameters only if it's enabled.
	if s.stos.Spec.CSI.EnableControllerExpandCreds {
		parameters[csiControllerExpandSecretNameKey] = csiControllerExpandSecretName
		parameters[csiControllerExpandSecretnamespaceKey] = s.stos.Spec.GetResourceNS()
	}

	return s.k8sResourceManager.StorageClass(s.stos.Spec.GetStorageClassName(), nil, StorageOSProvisionerName, parameters).Create()
}
