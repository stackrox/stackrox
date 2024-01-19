package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/pkg/manifest"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func log(msg string, params ...interface{}) {
	fmt.Printf(msg+"\n", params...)
}

func main() {
	var kubeconfig *string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "(optional) absolute path to the kubeconfig file")
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	ctx := context.Background()

	err = manifest.ApplyCentral(ctx, clientset)

	if err != nil {
		panic(err)
	}
}
