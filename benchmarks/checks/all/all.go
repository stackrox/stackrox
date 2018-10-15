package all

import (
	_ "github.com/stackrox/rox/benchmarks/checks/configuration_files"                     // Checks for Docker configuration files
	_ "github.com/stackrox/rox/benchmarks/checks/container_images_and_build"              // Checks for Docker images
	_ "github.com/stackrox/rox/benchmarks/checks/container_runtime"                       // Checks for running Docker containers
	_ "github.com/stackrox/rox/benchmarks/checks/docker_daemon_configuration"             // Checks for Docker daemon
	_ "github.com/stackrox/rox/benchmarks/checks/docker_security_operations"              // Checks for Docker security
	_ "github.com/stackrox/rox/benchmarks/checks/host_configuration"                      // Checks for Docker hosts
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/federated/api_server"         // Checks for Docker Swarm
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/federated/controller_manager" // Checks for Kubernetes federated controller manager
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/master/api_server"            // Checks for Kubernetes master api server
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/master/configuration_files"   // Checks for Kubernetes master configuration files
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/master/controller_manager"    // Checks for Kubernetes master controller manager
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/master/etcd"                  // Checks for Kubernetes etcd
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/master/scheduler"             // Checks for Kubernetes master scheduler
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/master/security_primitives"   // Checks for Kubernetes security primitives
	_ "github.com/stackrox/rox/benchmarks/checks/kubernetes/worker/kubelet"               // Checks for Kubernetes worker Kubelet
	_ "github.com/stackrox/rox/benchmarks/checks/swarm"                                   // Checks for Kubernetes federated api server
)
