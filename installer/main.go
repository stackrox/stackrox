package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/stackrox/rox/installer/manifest"
)

func log(msg string, params ...interface{}) {
	fmt.Printf(msg+"\n", params...)
}

func main() {
	configPath := flag.String("conf", "./installer.yaml", "Path to installer's configuration file.")
	// kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "(optional) absolute path to the kubeconfig file")
	// kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	kubeconfigFlag := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	action := flag.Arg(0)
	generatorSet := flag.Arg(1)

	var config *rest.Config
	var err error

	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig == "" {
		kubeconfig = *kubeconfigFlag
	}

	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			home := homedir.HomeDir()
			config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
			if err != nil {
				println(err.Error())
				return
			}
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			println(err.Error())
			return
		}
	}

	cfg, err := manifest.ReadConfig(*configPath)
	if err != nil {
		fmt.Printf("failed to load configuration %q: %v\n", *configPath, err)
		return
	}

	cfg.Action = action

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		println(err.Error())
		return
	}

	ctx := context.Background()

	m, err := manifest.New(cfg, clientset, config)
	if err != nil {
		println(err.Error())
		return
	}

	set, found := manifest.GeneratorSets[generatorSet]
	if !found {
		fmt.Printf("Invalid set '%s'. Valid options are central, securedcluster, or crs\n", generatorSet)
		return
	}

	switch action {
	case "apply":
		if err = m.Apply(ctx, *set); err != nil {
			println(err.Error())
			return
		}
	case "export":
		if err = m.Export(ctx, *set); err != nil {
			println(err.Error())
			return
		}
	}
}
