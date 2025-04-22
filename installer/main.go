package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/stackrox/rox/installer/manifest"
)

func log(msg string, params ...interface{}) {
	fmt.Printf(msg+"\n", params...)
}

func main() {
	configPath := flag.String("conf", "./installer.yaml", "Path to installer's configuration file.")
	var kubeconfig *string
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "(optional) absolute path to the kubeconfig file")
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	action := flag.Arg(0)
	generatorSet := flag.Arg(1)

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		println(err.Error())
		return
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
