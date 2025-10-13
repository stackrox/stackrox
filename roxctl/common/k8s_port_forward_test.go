package common

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_getCentralAPIPort(t *testing.T) {
	type testCase struct {
		pod          *corev1.Pod
		expectedPort int32
	}
	tests := map[string]testCase{
		"good 1": {
			pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{
				Name: "central",
				Ports: []corev1.ContainerPort{
					{Name: "api", ContainerPort: 1234},
				},
			}}}},
			expectedPort: 1234,
		},
		"good 2": {
			pod: &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{
				{},
				{
					Name: "central",
					Ports: []corev1.ContainerPort{
						{Name: "http", ContainerPort: 80},
						{Name: "api", ContainerPort: 1234},
					},
				}}}},
			expectedPort: 1234,
		},
	}
	for name, tt := range tests {
		assert.Equal(t, int32(tt.expectedPort), getCentralAPIPort(tt.pod), name)
	}
}

func Test_getCentralPod(t *testing.T) {
	labels := map[string]string{"app": "central"}
	const ns = "test_ns"
	centralService := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{Namespace: ns, Name: "central"},
		Spec:       corev1.ServiceSpec{Selector: labels},
	}
	centralPod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{Namespace: ns, Labels: labels},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name:  "central",
			Ports: []corev1.ContainerPort{{Name: "api", ContainerPort: 1234}},
		}}}}
	tests := map[string]struct {
		objs    []runtime.Object
		want    *corev1.Pod
		wantErr bool
	}{
		"good": {
			objs: []runtime.Object{centralService, centralPod},
			want: centralPod,
		},
		"no service": {
			objs:    []runtime.Object{centralPod},
			wantErr: true,
		},
		"no pod": {
			objs:    []runtime.Object{centralService},
			wantErr: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			core := fake.NewSimpleClientset(tt.objs...).CoreV1()

			got, err := getCentralPod(context.Background(), core, ns)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCentralPod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getCentralPod() = %v, want %v", got, tt.want)
			}
		})
	}
}
