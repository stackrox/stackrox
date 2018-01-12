package all

import (
	// Checks for Docker configuration files
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/configuration_files"
	// Checks for Docker images
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/container_images_and_build"
	// Checks for running Docker containers
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/container_runtime"
	// Checks for Docker daemon
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/docker_daemon_configuration"
	// Checks for Docker security
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/docker_security_operations"
	// Checks for Docker hosts
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/host_configuration"
	// Checks for Docker Swarm
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/swarm"
	// Checks for Kubernetes federated api server
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/federated/api_server"
	// Checks for Kubernetes federated controller manager
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/federated/controller_manager"
	// Checks for Kubernetes master api server
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/master/api_server"
	// Checks for Kubernetes master configuration files
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/master/configuration_files"
	// Checks for Kubernetes master controller manager
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/master/controller_manager"
	// Checks for Kubernetes etcd
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/master/etcd"
	// Checks for Kubernetes master scheduler
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/master/scheduler"
	// Checks for Kubernetes security primitives
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/master/security_primitives"
	// Checks for Kubernetes worker Kubelet
	_ "bitbucket.org/stack-rox/apollo/pkg/checks/kubernetes/worker/kubelet"
)
