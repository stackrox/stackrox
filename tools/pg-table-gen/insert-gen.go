package main

import (
	"database/sql"

	"github.com/stackrox/rox/generated/storage"
)

const (
	insertdeploymentQuery = `INSERT INTO Deployment (id, name, hash, type, namespace, namespaceid, orchestratorcomponent, replicas, labels, podlabels, labelselector_matchlabels, created, clusterid, clustername, annotations, priority, inactive, imagepullsecrets, serviceaccount, serviceaccountpermissionlevel, automountserviceaccounttoken, hostnetwork, hostpid, hostipc, statetimestamp, riskscore, processtags) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)`
)

func insertDeployment(tx *sql.Tx, deployment *storage.Deployment) error {
	if _, err := tx.Exec(insertdeploymentQuery,

		deployment.GetId(),
		deployment.GetName(),
		deployment.GetHash(),
		deployment.GetType(),
		deployment.GetNamespace(),
		deployment.GetNamespaceId(),
		deployment.GetOrchestratorComponent(),
		deployment.GetReplicas(),
		deployment.GetLabels(),
		deployment.GetPodLabels(),
		deployment.GetLabelSelector().GetMatchLabels(),
		deployment.GetCreated(),
		deployment.GetClusterId(),
		deployment.GetClusterName(),
		deployment.GetAnnotations(),
		deployment.GetPriority(),
		deployment.GetInactive(),
		deployment.GetImagePullSecrets(),
		deployment.GetServiceAccount(),
		deployment.GetServiceAccountPermissionLevel(),
		deployment.GetAutomountServiceAccountToken(),
		deployment.GetHostNetwork(),
		deployment.GetHostPid(),
		deployment.GetHostIpc(),
		deployment.GetStateTimestamp(),
		deployment.GetRiskScore(),
		deployment.GetProcessTags(),
	); err != nil {
		return err
	}

	for _, val := range deployment.GetLabelSelector().GetRequirements() {
		if err := insertDeployment_LabelSelector_Requirements(tx, val); err != nil {
			return err
		}
	}
	for _, val := range deployment.GetContainers() {
		if err := insertDeployment_Containers(tx, val); err != nil {
			return err
		}
	}
	for _, val := range deployment.GetTolerations() {
		if err := insertDeployment_Tolerations(tx, val); err != nil {
			return err
		}
	}
	for _, val := range deployment.GetPorts() {
		if err := insertDeployment_Ports(tx, val); err != nil {
			return err
		}
	}
	return nil
}

const (
	insertdeployment_labelselector_requirementsQuery = `INSERT INTO Deployment_LabelSelector.Requirements (key, op, values) VALUES($1, $2, $3)`
)

func insertDeployment_LabelSelector_Requirements(tx *sql.Tx, deployment_labelselector_requirements *storage.LabelSelector_Requirement) error {
	if _, err := tx.Exec(insertdeployment_labelselector_requirementsQuery,

		deployment_labelselector_requirements.GetKey(),
		deployment_labelselector_requirements.GetOp(),
		deployment_labelselector_requirements.GetValues(),
	); err != nil {
		return err
	}

	return nil
}

const (
	insertdeployment_containersQuery = `INSERT INTO Deployment_Containers (id, config_command, config_args, config_directory, config_user, config_uid, config_apparmorprofile, image_id, image_name_registry, image_name_remote, image_name_tag, image_name_fullname, image_notpullable, securitycontext_privileged, securitycontext_selinux_user, securitycontext_selinux_role, securitycontext_selinux_type, securitycontext_selinux_level, securitycontext_dropcapabilities, securitycontext_addcapabilities, securitycontext_readonlyrootfilesystem, securitycontext_seccompprofile_type, securitycontext_seccompprofile_localhostprofile, resources_cpucoresrequest, resources_cpucoreslimit, resources_memorymbrequest, resources_memorymblimit, name) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)`
)

func insertDeployment_Containers(tx *sql.Tx, deployment_containers *storage.Container) error {
	if _, err := tx.Exec(insertdeployment_containersQuery,

		deployment_containers.GetId(),
		deployment_containers.GetConfig().GetCommand(),
		deployment_containers.GetConfig().GetArgs(),
		deployment_containers.GetConfig().GetDirectory(),
		deployment_containers.GetConfig().GetUser(),
		deployment_containers.GetConfig().GetUid(),
		deployment_containers.GetConfig().GetAppArmorProfile(),
		deployment_containers.GetImage().GetId(),
		deployment_containers.GetImage().GetName().GetRegistry(),
		deployment_containers.GetImage().GetName().GetRemote(),
		deployment_containers.GetImage().GetName().GetTag(),
		deployment_containers.GetImage().GetName().GetFullName(),
		deployment_containers.GetImage().GetNotPullable(),
		deployment_containers.GetSecurityContext().GetPrivileged(),
		deployment_containers.GetSecurityContext().GetSelinux().GetUser(),
		deployment_containers.GetSecurityContext().GetSelinux().GetRole(),
		deployment_containers.GetSecurityContext().GetSelinux().GetType(),
		deployment_containers.GetSecurityContext().GetSelinux().GetLevel(),
		deployment_containers.GetSecurityContext().GetDropCapabilities(),
		deployment_containers.GetSecurityContext().GetAddCapabilities(),
		deployment_containers.GetSecurityContext().GetReadOnlyRootFilesystem(),
		deployment_containers.GetSecurityContext().GetSeccompProfile().GetType(),
		deployment_containers.GetSecurityContext().GetSeccompProfile().GetLocalhostProfile(),
		deployment_containers.GetResources().GetCpuCoresRequest(),
		deployment_containers.GetResources().GetCpuCoresLimit(),
		deployment_containers.GetResources().GetMemoryMbRequest(),
		deployment_containers.GetResources().GetMemoryMbLimit(),
		deployment_containers.GetName(),
	); err != nil {
		return err
	}

	for _, val := range deployment_containers.GetConfig().GetEnv() {
		if err := insertDeployment_Containers_Config_Env(tx, val); err != nil {
			return err
		}
	}
	for _, val := range deployment_containers.GetVolumes() {
		if err := insertDeployment_Containers_Volumes(tx, val); err != nil {
			return err
		}
	}
	for _, val := range deployment_containers.GetPorts() {
		if err := insertDeployment_Containers_Ports(tx, val); err != nil {
			return err
		}
	}
	for _, val := range deployment_containers.GetSecrets() {
		if err := insertDeployment_Containers_Secrets(tx, val); err != nil {
			return err
		}
	}
	return nil
}

const (
	insertdeployment_containers_config_envQuery = `INSERT INTO Deployment_Containers_Config.Env (key, value, envvarsource) VALUES($1, $2, $3)`
)

func insertDeployment_Containers_Config_Env(tx *sql.Tx, deployment_containers_config_env *storage.ContainerConfig_EnvironmentConfig) error {
	if _, err := tx.Exec(insertdeployment_containers_config_envQuery,

		deployment_containers_config_env.GetKey(),
		deployment_containers_config_env.GetValue(),
		deployment_containers_config_env.GetEnvVarSource(),
	); err != nil {
		return err
	}

	return nil
}

const (
	insertdeployment_containers_volumesQuery = `INSERT INTO Deployment_Containers_Volumes (name, source, destination, readonly, type, mountpropagation) VALUES($1, $2, $3, $4, $5, $6)`
)

func insertDeployment_Containers_Volumes(tx *sql.Tx, deployment_containers_volumes *storage.Volume) error {
	if _, err := tx.Exec(insertdeployment_containers_volumesQuery,

		deployment_containers_volumes.GetName(),
		deployment_containers_volumes.GetSource(),
		deployment_containers_volumes.GetDestination(),
		deployment_containers_volumes.GetReadOnly(),
		deployment_containers_volumes.GetType(),
		deployment_containers_volumes.GetMountPropagation(),
	); err != nil {
		return err
	}

	return nil
}

const (
	insertdeployment_containers_portsQuery = `INSERT INTO Deployment_Containers_Ports (name, containerport, protocol, exposure, exposedport) VALUES($1, $2, $3, $4, $5)`
)

func insertDeployment_Containers_Ports(tx *sql.Tx, deployment_containers_ports *storage.PortConfig) error {
	if _, err := tx.Exec(insertdeployment_containers_portsQuery,

		deployment_containers_ports.GetName(),
		deployment_containers_ports.GetContainerPort(),
		deployment_containers_ports.GetProtocol(),
		deployment_containers_ports.GetExposure(),
		deployment_containers_ports.GetExposedPort(),
	); err != nil {
		return err
	}

	for _, val := range deployment_containers_ports.GetExposureInfos() {
		if err := insertDeployment_Containers_Ports_ExposureInfos(tx, val); err != nil {
			return err
		}
	}
	return nil
}

const (
	insertdeployment_containers_ports_exposureinfosQuery = `INSERT INTO Deployment_Containers_Ports_ExposureInfos (level, servicename, serviceid, serviceclusterip, serviceport, nodeport, externalips, externalhostnames) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`
)

func insertDeployment_Containers_Ports_ExposureInfos(tx *sql.Tx, deployment_containers_ports_exposureinfos *storage.PortConfig_ExposureInfo) error {
	if _, err := tx.Exec(insertdeployment_containers_ports_exposureinfosQuery,

		deployment_containers_ports_exposureinfos.GetLevel(),
		deployment_containers_ports_exposureinfos.GetServiceName(),
		deployment_containers_ports_exposureinfos.GetServiceId(),
		deployment_containers_ports_exposureinfos.GetServiceClusterIp(),
		deployment_containers_ports_exposureinfos.GetServicePort(),
		deployment_containers_ports_exposureinfos.GetNodePort(),
		deployment_containers_ports_exposureinfos.GetExternalIps(),
		deployment_containers_ports_exposureinfos.GetExternalHostnames(),
	); err != nil {
		return err
	}

	return nil
}

const (
	insertdeployment_containers_secretsQuery = `INSERT INTO Deployment_Containers_Secrets (name, path) VALUES($1, $2)`
)

func insertDeployment_Containers_Secrets(tx *sql.Tx, deployment_containers_secrets *storage.EmbeddedSecret) error {
	if _, err := tx.Exec(insertdeployment_containers_secretsQuery,

		deployment_containers_secrets.GetName(),
		deployment_containers_secrets.GetPath(),
	); err != nil {
		return err
	}

	return nil
}

const (
	insertdeployment_tolerationsQuery = `INSERT INTO Deployment_Tolerations (key, operator, value, tainteffect) VALUES($1, $2, $3, $4)`
)

func insertDeployment_Tolerations(tx *sql.Tx, deployment_tolerations *storage.Toleration) error {
	if _, err := tx.Exec(insertdeployment_tolerationsQuery,

		deployment_tolerations.GetKey(),
		deployment_tolerations.GetOperator(),
		deployment_tolerations.GetValue(),
		deployment_tolerations.GetTaintEffect(),
	); err != nil {
		return err
	}

	return nil
}

const (
	insertdeployment_portsQuery = `INSERT INTO Deployment_Ports (name, containerport, protocol, exposure, exposedport) VALUES($1, $2, $3, $4, $5)`
)

func insertDeployment_Ports(tx *sql.Tx, deployment_ports *storage.PortConfig) error {
	if _, err := tx.Exec(insertdeployment_portsQuery,
		deployment_ports.GetName(),
		deployment_ports.GetContainerPort(),
		deployment_ports.GetProtocol(),
		deployment_ports.GetExposure(),
		deployment_ports.GetExposedPort(),
	); err != nil {
		return err
	}

	for _, val := range deployment_ports.GetExposureInfos() {
		if err := insertDeployment_Ports_ExposureInfos(tx, val); err != nil {
			return err
		}
	}
	return nil
}

const (
	insertdeployment_ports_exposureinfosQuery = `INSERT INTO Deployment_Ports_ExposureInfos (level, servicename, serviceid, serviceclusterip, serviceport, nodeport, externalips, externalhostnames) VALUES($1, $2, $3, $4, $5, $6, $7, $8)`
)

func insertDeployment_Ports_ExposureInfos(tx *sql.Tx, deployment_ports_exposureinfos *storage.PortConfig_ExposureInfo) error {
	if _, err := tx.Exec(insertdeployment_ports_exposureinfosQuery,
		deployment_ports_exposureinfos.GetLevel(),
		deployment_ports_exposureinfos.GetServiceName(),
		deployment_ports_exposureinfos.GetServiceId(),
		deployment_ports_exposureinfos.GetServiceClusterIp(),
		deployment_ports_exposureinfos.GetServicePort(),
		deployment_ports_exposureinfos.GetNodePort(),
		deployment_ports_exposureinfos.GetExternalIps(),
		deployment_ports_exposureinfos.GetExternalHostnames(),
	); err != nil {
		return err
	}
	return nil
}
