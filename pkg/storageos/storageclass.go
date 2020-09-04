package storageos

func (s *Deployment) createStorageClass() error {
	parameters := map[string]string{
		"pool": "default",
	}

	if s.stos.Spec.CSI.Enable {
		// Add CSI creds secrets in parameters.
		if CSIV1Supported(s.k8sVersion) {
			// New CSI secret parameter keys were introduced in CSI v1.
			parameters[csiV1FSType] = defaultFSType
			if s.stos.Spec.CSI.EnableProvisionCreds {
				parameters[csiV1ProvisionerSecretNameKey] = csiProvisionerSecretName
				parameters[csiV1ProvisionerSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
			}
			if s.stos.Spec.CSI.EnableControllerPublishCreds {
				parameters[csiV1ControllerPublishSecretNameKey] = csiControllerPublishSecretName
				parameters[csiV1ControllerPublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
			}
			if s.stos.Spec.CSI.EnableNodePublishCreds {
				parameters[csiV1NodePublishSecretNameKey] = csiNodePublishSecretName
				parameters[csiV1NodePublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
			}
			// Add expand parameters only if it's supported.
			if CSIExternalResizerSupported(s.k8sVersion) {
				if s.stos.Spec.CSI.EnableControllerExpandCreds {
					parameters[csiV1ControllerExpandSecretNameKey] = csiControllerExpandSecretName
					parameters[csiV1ControllerExpandSecretnamespaceKey] = s.stos.Spec.GetResourceNS()
				}
			}
		} else {
			parameters[fsType] = defaultFSType
			if s.stos.Spec.CSI.EnableProvisionCreds {
				parameters[csiV0ProvisionerSecretNameKey] = csiProvisionerSecretName
				parameters[csiV0ProvisionerSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
			}
			if s.stos.Spec.CSI.EnableControllerPublishCreds {
				parameters[csiV0ControllerPublishSecretNameKey] = csiControllerPublishSecretName
				parameters[csiV0ControllerPublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
			}
			if s.stos.Spec.CSI.EnableNodePublishCreds {
				parameters[csiV0NodePublishSecretNameKey] = csiNodePublishSecretName
				parameters[csiV0NodePublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
			}
		}
	} else {
		parameters[fsType] = defaultFSType
		// Add StorageOS admin secrets name and namespace.
		parameters[secretNamespaceKey] = s.stos.Spec.SecretRefNamespace
		parameters[secretNameKey] = s.stos.Spec.SecretRefName
	}

	return s.k8sResourceManager.StorageClass(s.stos.Spec.GetStorageClassName(), nil, StorageOSProvisionerName, parameters).Create()
}
