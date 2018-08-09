package all

import (
	// Checks for Docker configuration files
	_ "github.com/stackrox/rox/pkg/checks/configuration_files"
	// Checks for Docker images
	_ "github.com/stackrox/rox/pkg/checks/container_images_and_build"
	// Checks for running Docker containers
	_ "github.com/stackrox/rox/pkg/checks/container_runtime"
	// Checks for Docker daemon
	_ "github.com/stackrox/rox/pkg/checks/docker_daemon_configuration"
	// Checks for Docker security
	_ "github.com/stackrox/rox/pkg/checks/docker_security_operations"
	// Checks for Docker hosts
	_ "github.com/stackrox/rox/pkg/checks/host_configuration"
	// Checks for Docker Swarm
	_ "github.com/stackrox/rox/pkg/checks/swarm"
	// Checks for Kubernetes federated api server
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/federated/api_server"
	// Checks for Kubernetes federated controller manager
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/federated/controller_manager"
	// Checks for Kubernetes master api server
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/master/api_server"
	// Checks for Kubernetes master configuration files
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/master/configuration_files"
	// Checks for Kubernetes master controller manager
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/master/controller_manager"
	// Checks for Kubernetes etcd
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/master/etcd"
	// Checks for Kubernetes master scheduler
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/master/scheduler"
	// Checks for Kubernetes security primitives
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/master/security_primitives"
	// Checks for Kubernetes worker Kubelet
	_ "github.com/stackrox/rox/pkg/checks/kubernetes/worker/kubelet"
)
