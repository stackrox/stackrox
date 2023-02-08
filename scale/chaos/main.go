package main

import (
	"bytes"
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	log = logging.LoggerForModule()
)

func applyJitter(t time.Duration) time.Duration {
	multiplier := rand.Float32() + 0.5
	return time.Duration(multiplier * float32(t))
}

type killFunc func(client *kubernetes.Clientset, config *rest.Config, pods []corev1.Pod)

func execKill(client *kubernetes.Clientset, config *rest.Config, pods []corev1.Pod) {
	signals := []string{"-15"}
	signal := signals[rand.Intn(len(signals))]
	cmd := []string{"kill", signal, "1"}
	for _, pod := range pods {
		log.Infof("Exec'ing into pod %s and running %+v", pod.Name, cmd)

		req := client.CoreV1().RESTClient().Post().Resource("pods").Name(pod.Name).
			Namespace("stackrox").SubResource("exec")
		option := &corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}
		req.VersionedParams(
			option,
			scheme.ParameterCodec,
		)
		exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
		if err != nil {
			log.Errorf("error executing spdy: %v", err)
			return
		}
		var stdout, stderr bytes.Buffer
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: &stdout,
			Stderr: &stderr,
		})
		if err != nil {
			log.Errorf("Streaming: %v", err)
			return
		}
		log.Infof("Output: %s %s", stdout.Bytes(), stderr.Bytes())
	}
}

func getPods(client *kubernetes.Clientset, labelSelector string) []corev1.Pod {
	ctx := context.Background()
	podList, err := client.CoreV1().Pods("stackrox").List(ctx, v1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		log.Panicf("error listing pods: %v", err)
	}
	return podList.Items
}

func podKill(client *kubernetes.Clientset, _ *rest.Config, pods []corev1.Pod) {
	for _, pod := range pods {
		gracePeriod := rand.Int63n(10)
		log.Infof("Deleting pod %s with grace period %d", pod.Name, gracePeriod)
		err := client.CoreV1().Pods("stackrox").Delete(context.Background(), pod.Name, v1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
		})
		if err != nil {
			log.Errorf("error deleting pod %s: %v", pod.Name, err)
		}
	}
}

func selectOption() bool {
	return rand.Float32() < 0.5
}

func main() {
	// kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	//config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	//if err != nil {
	//	panic(err.Error())
	//}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Panicf("obtaining in-cluster Kubernetes config: %v", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("creating Kubernetes clientset: %v", err)
	}

	log.Info("Successfully initialized Kubernetes client")

	signalsC := make(chan os.Signal, 1)
	signal.Notify(signalsC, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	interval := env.ChaosIntervalEnv.DurationSetting()
	nextInterval := applyJitter(interval)
	log.Infof("Will attempt to terminate Central in %0.2f seconds", nextInterval.Seconds())

	for {
		select {
		case sig := <-signalsC:
			log.Infof("Caught %s signal", sig)
			log.Info("Chaos monkey terminated")
			return
		case <-time.After(nextInterval):
			nextInterval = applyJitter(interval)

			var killFn = podKill
			var selector string
			if selectOption() {
				selector = "app=central"
			} else {
				selector = "app=central-db"
			}
			pods := getPods(client, selector)
			killFn(client, config, pods)
		}
	}
}
