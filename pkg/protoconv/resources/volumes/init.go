package volumes

// RegisterAll registers all volume converters into the VolumeRegistry.
// This must be called before using protoconv for Kubernetes volumes.
func RegisterAll() {
	VolumeRegistry[azureDiskType] = createAzureDisk
	VolumeRegistry[azureFileType] = createAzureFile
	VolumeRegistry[cephFSType] = createCephfs
	VolumeRegistry[cinderType] = createCinder
	VolumeRegistry[configMapType] = createConfigMap
	VolumeRegistry[ebsType] = createEBS
	VolumeRegistry[emptyDirType] = createEmptyDir
	VolumeRegistry[gcePersistentDiskType] = createGCEPersistentDisk
	VolumeRegistry[gitRepoType] = createGitRepo
	VolumeRegistry[glusterfsType] = createGlusterfs
	VolumeRegistry[hostPathType] = createHostPath
	VolumeRegistry[nfsType] = createNFS
	VolumeRegistry[persistentVolumeClaimType] = createPersistentVolumeClaim
	VolumeRegistry[rbdType] = createRBD
	VolumeRegistry[secretType] = createSecret
}
