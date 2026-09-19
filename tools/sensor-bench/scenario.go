package main

import (
	"context"
	"fmt"
	"math/rand"
	"runtime/pprof"

	"github.com/stackrox/rox/sensor/debugger/k8s"
	v1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func RunScenario(ctx context.Context, h *Harness, cfg *Config) error {
	labels := pprof.Labels("component", "bench-harness")
	pprof.Do(ctx, labels, func(ctx context.Context) {
		injectWorkload(ctx, h.FakeClient, cfg)
	})
	return nil
}

func injectWorkload(ctx context.Context, client *k8s.ClientSet, cfg *Config) {
	w := cfg.Workload

	namespaces := make([]string, w.Namespaces)
	for i := range namespaces {
		namespaces[i] = fmt.Sprintf("bench-ns-%d", i)
		createNamespace(client, namespaces[i])
	}

	roleNames := make([]string, w.Roles.Count)
	for i := range w.Roles.Count {
		ns := namespaces[i%len(namespaces)]
		roleName := fmt.Sprintf("role-%d", i)
		roleNames[i] = roleName
		if w.Roles.ClusterWide {
			createClusterRole(client, roleName, w.Roles.Verbs)
		} else {
			createRole(client, ns, roleName, w.Roles.Verbs)
		}
	}

	for i := range w.RoleBindings.Count {
		ns := namespaces[i%len(namespaces)]
		bindingName := fmt.Sprintf("binding-%d", i)
		roleName := roleNames[i%len(roleNames)]
		sa := serviceAccountFor(w.Deployments.ServiceAccount, i)
		if w.Roles.ClusterWide {
			createClusterRoleBinding(client, bindingName, roleName, ns, sa)
		} else {
			createRoleBinding(client, ns, bindingName, roleName, sa)
		}
	}

	for i := range w.Services.Count {
		ns := namespaces[i%len(namespaces)]
		serviceName := fmt.Sprintf("svc-%d", i)
		appLabel := fmt.Sprintf("app-%d", i)
		createService(client, ns, serviceName, appLabel, w.Services.Type)
	}

	for i := range w.Deployments.Count {
		ns := namespaces[i%len(namespaces)]
		deploymentName := fmt.Sprintf("deploy-%d", i)
		appLabel := fmt.Sprintf("app-%d", i)
		sa := serviceAccountFor(w.Deployments.ServiceAccount, i)
		createDeployment(client, ns, deploymentName, appLabel, sa, w.Deployments.Image, w.Deployments.Containers)
	}
}

func serviceAccountFor(saConfig string, index int) string {
	if saConfig == "random" {
		return fmt.Sprintf("sa-%s", randString(5))
	}
	return saConfig
}

func createNamespace(client *k8s.ClientSet, name string) {
	_, err := client.Kubernetes().CoreV1().Namespaces().Create(context.Background(), &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating namespace %s: %v", name, err))
	}
}

func createRole(client *k8s.ClientSet, namespace, name string, verbs []string) {
	_, err := client.Kubernetes().RbacV1().Roles(namespace).Create(context.Background(), &rbac.Role{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Rules: []rbac.PolicyRule{{
			Verbs:     verbs,
			APIGroups: []string{"*"},
			Resources: []string{"*"},
		}},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating role %s/%s: %v", namespace, name, err))
	}
}

func createClusterRole(client *k8s.ClientSet, name string, verbs []string) {
	_, err := client.Kubernetes().RbacV1().ClusterRoles().Create(context.Background(), &rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Rules: []rbac.PolicyRule{{
			Verbs:     verbs,
			APIGroups: []string{"*"},
			Resources: []string{"*"},
		}},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating cluster role %s: %v", name, err))
	}
}

func createRoleBinding(client *k8s.ClientSet, namespace, name, roleName, serviceAccount string) {
	_, err := client.Kubernetes().RbacV1().RoleBindings(namespace).Create(context.Background(), &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccount,
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{Kind: "Role", Name: roleName, APIGroup: "rbac.authorization.k8s.io"},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating role binding %s/%s: %v", namespace, name, err))
	}
}

func createClusterRoleBinding(client *k8s.ClientSet, name, roleName, namespace, serviceAccount string) {
	_, err := client.Kubernetes().RbacV1().ClusterRoleBindings().Create(context.Background(), &rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Subjects: []rbac.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccount,
			Namespace: namespace,
		}},
		RoleRef: rbac.RoleRef{Kind: "ClusterRole", Name: roleName, APIGroup: "rbac.authorization.k8s.io"},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating cluster role binding %s: %v", name, err))
	}
}

func createService(client *k8s.ClientSet, namespace, name, appLabel, svcType string) {
	serviceType := core.ServiceTypeClusterIP
	switch svcType {
	case "NodePort":
		serviceType = core.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = core.ServiceTypeLoadBalancer
	}
	_, err := client.Kubernetes().CoreV1().Services(namespace).Create(context.Background(), &core.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.IntOrString{IntVal: 8080},
			}},
			Selector: map[string]string{"app": appLabel},
			Type:     serviceType,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating service %s/%s: %v", namespace, name, err))
	}
}

func createDeployment(client *k8s.ClientSet, namespace, name, appLabel, serviceAccount, image string, numContainers int) {
	containers := make([]core.Container, numContainers)
	for i := range containers {
		containers[i] = core.Container{
			Name:  fmt.Sprintf("container-%d", i),
			Image: image,
		}
	}

	_, err := client.Kubernetes().AppsV1().Deployments(namespace).Create(context.Background(), &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: v1.DeploymentSpec{
			Template: core.PodTemplateSpec{
				Spec: core.PodSpec{
					Containers:         containers,
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		panic(fmt.Sprintf("creating deployment %s/%s: %v", namespace, name, err))
	}
}
