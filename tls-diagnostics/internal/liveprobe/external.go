package liveprobe

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type externalEndpoint struct {
	Address string
	Method  string // "LoadBalancer", "NodePort"
}

func detectExternalEndpoint(ctx context.Context, client kubernetes.Interface, namespace string) *externalEndpoint {
	svc, err := client.CoreV1().Services(namespace).Get(ctx, "central-loadbalancer", metav1.GetOptions{})
	if err != nil {
		return nil
	}

	switch svc.Spec.Type {
	case "LoadBalancer":
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				port := 443
				if len(svc.Spec.Ports) > 0 {
					port = int(svc.Spec.Ports[0].Port)
				}
				return &externalEndpoint{
					Address: fmt.Sprintf("%s:%d", ingress.IP, port),
					Method:  "LoadBalancer",
				}
			}
			if ingress.Hostname != "" {
				port := 443
				if len(svc.Spec.Ports) > 0 {
					port = int(svc.Spec.Ports[0].Port)
				}
				return &externalEndpoint{
					Address: fmt.Sprintf("%s:%d", ingress.Hostname, port),
					Method:  "LoadBalancer",
				}
			}
		}
	case "NodePort":
		if len(svc.Spec.Ports) > 0 && svc.Spec.Ports[0].NodePort != 0 {
			return &externalEndpoint{
				Address: fmt.Sprintf("localhost:%d", svc.Spec.Ports[0].NodePort),
				Method:  "NodePort",
			}
		}
	}

	return nil
}
