package types

import "github.com/stackrox/rox/generated/storage"

// ToStoredDeployment converts a storage.Deployment (API type) to a storage.StoredDeployment (storage type).
// It merges containers, init_containers, and ephemeral_containers into a single containers list
// with ContainerType set on each StoredContainer.
func ToStoredDeployment(d *storage.Deployment) *storage.StoredDeployment {
	if d == nil {
		return nil
	}

	sd := &storage.StoredDeployment{
		Id:                            d.GetId(),
		Name:                          d.GetName(),
		Hash:                          d.GetHash(),
		Type:                          d.GetType(),
		Namespace:                     d.GetNamespace(),
		NamespaceId:                   d.GetNamespaceId(),
		OrchestratorComponent:         d.GetOrchestratorComponent(),
		Replicas:                      d.GetReplicas(),
		Labels:                        d.GetLabels(),
		PodLabels:                     d.GetPodLabels(),
		LabelSelector:                 d.GetLabelSelector(),
		Created:                       d.GetCreated(),
		ClusterId:                     d.GetClusterId(),
		ClusterName:                   d.GetClusterName(),
		Annotations:                   d.GetAnnotations(),
		Priority:                      d.GetPriority(),
		Inactive:                      d.GetInactive(),
		ImagePullSecrets:              d.GetImagePullSecrets(),
		ServiceAccount:                d.GetServiceAccount(),
		ServiceAccountPermissionLevel: d.GetServiceAccountPermissionLevel(),
		AutomountServiceAccountToken:  d.GetAutomountServiceAccountToken(),
		HostNetwork:                   d.GetHostNetwork(),
		HostPid:                       d.GetHostPid(),
		HostIpc:                       d.GetHostIpc(),
		RuntimeClass:                  d.GetRuntimeClass(),
		Tolerations:                   d.GetTolerations(),
		Ports:                         d.GetPorts(),
		StateTimestamp:                d.GetStateTimestamp(),
		RiskScore:                     d.GetRiskScore(),
		PlatformComponent:             d.GetPlatformComponent(),
	}

	// Merge all container types into a single list with ContainerType set.
	var containers []*storage.StoredContainer
	for _, c := range d.GetContainers() {
		containers = append(containers, toStoredContainer(c, storage.ContainerType_STANDARD))
	}
	for _, c := range d.GetInitContainers() {
		containers = append(containers, toStoredContainer(c, storage.ContainerType_INIT))
	}
	for _, c := range d.GetEphemeralContainers() {
		containers = append(containers, toStoredContainer(c, storage.ContainerType_EPHEMERAL))
	}
	sd.Containers = containers

	return sd
}

// FromStoredDeployment converts a storage.StoredDeployment (storage type) to a storage.Deployment (API type).
// It splits the single containers list by ContainerType into containers, init_containers, and ephemeral_containers.
func FromStoredDeployment(sd *storage.StoredDeployment) *storage.Deployment {
	if sd == nil {
		return nil
	}

	d := &storage.Deployment{
		Id:                            sd.GetId(),
		Name:                          sd.GetName(),
		Hash:                          sd.GetHash(),
		Type:                          sd.GetType(),
		Namespace:                     sd.GetNamespace(),
		NamespaceId:                   sd.GetNamespaceId(),
		OrchestratorComponent:         sd.GetOrchestratorComponent(),
		Replicas:                      sd.GetReplicas(),
		Labels:                        sd.GetLabels(),
		PodLabels:                     sd.GetPodLabels(),
		LabelSelector:                 sd.GetLabelSelector(),
		Created:                       sd.GetCreated(),
		ClusterId:                     sd.GetClusterId(),
		ClusterName:                   sd.GetClusterName(),
		Annotations:                   sd.GetAnnotations(),
		Priority:                      sd.GetPriority(),
		Inactive:                      sd.GetInactive(),
		ImagePullSecrets:              sd.GetImagePullSecrets(),
		ServiceAccount:                sd.GetServiceAccount(),
		ServiceAccountPermissionLevel: sd.GetServiceAccountPermissionLevel(),
		AutomountServiceAccountToken:  sd.GetAutomountServiceAccountToken(),
		HostNetwork:                   sd.GetHostNetwork(),
		HostPid:                       sd.GetHostPid(),
		HostIpc:                       sd.GetHostIpc(),
		RuntimeClass:                  sd.GetRuntimeClass(),
		Tolerations:                   sd.GetTolerations(),
		Ports:                         sd.GetPorts(),
		StateTimestamp:                sd.GetStateTimestamp(),
		RiskScore:                     sd.GetRiskScore(),
		PlatformComponent:             sd.GetPlatformComponent(),
	}

	// Split containers by type.
	for _, sc := range sd.GetContainers() {
		c := fromStoredContainer(sc)
		switch sc.GetContainerType() {
		case storage.ContainerType_INIT:
			d.InitContainers = append(d.InitContainers, c)
		case storage.ContainerType_EPHEMERAL:
			d.EphemeralContainers = append(d.EphemeralContainers, c)
		default:
			// STANDARD and UNSPECIFIED both map to regular containers.
			// Old data has UNSPECIFIED (0), treated as STANDARD.
			d.Containers = append(d.Containers, c)
		}
	}

	return d
}

func toStoredContainer(c *storage.Container, ct storage.ContainerType) *storage.StoredContainer {
	if c == nil {
		return nil
	}
	return &storage.StoredContainer{
		Id:              c.GetId(),
		Config:          c.GetConfig(),
		Image:           toStoredContainerImage(c.GetImage()),
		SecurityContext: c.GetSecurityContext(),
		Volumes:         c.GetVolumes(),
		Ports:           c.GetPorts(),
		Secrets:         c.GetSecrets(),
		Resources:       c.GetResources(),
		Name:            c.GetName(),
		LivenessProbe:   c.GetLivenessProbe(),
		ReadinessProbe:  c.GetReadinessProbe(),
		ContainerType:   ct,
	}
}

func fromStoredContainer(sc *storage.StoredContainer) *storage.Container {
	if sc == nil {
		return nil
	}
	return &storage.Container{
		Id:              sc.GetId(),
		Config:          sc.GetConfig(),
		Image:           fromStoredContainerImage(sc.GetImage()),
		SecurityContext: sc.GetSecurityContext(),
		Volumes:         sc.GetVolumes(),
		Ports:           sc.GetPorts(),
		Secrets:         sc.GetSecrets(),
		Resources:       sc.GetResources(),
		Name:            sc.GetName(),
		LivenessProbe:   sc.GetLivenessProbe(),
		ReadinessProbe:  sc.GetReadinessProbe(),
	}
}

func toStoredContainerImage(img *storage.ContainerImage) *storage.StoredContainerImage {
	if img == nil {
		return nil
	}
	return &storage.StoredContainerImage{
		Id:             img.GetId(),
		Name:           img.GetName(),
		NotPullable:    img.GetNotPullable(),
		IsClusterLocal: img.GetIsClusterLocal(),
		IdV2:           img.GetIdV2(),
	}
}

func fromStoredContainerImage(img *storage.StoredContainerImage) *storage.ContainerImage {
	if img == nil {
		return nil
	}
	return &storage.ContainerImage{
		Id:             img.GetId(),
		Name:           img.GetName(),
		NotPullable:    img.GetNotPullable(),
		IsClusterLocal: img.GetIsClusterLocal(),
		IdV2:           img.GetIdV2(),
	}
}
