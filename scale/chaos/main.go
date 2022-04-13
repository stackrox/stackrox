package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/logging"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	log = logging.LoggerForModule()

	gracePeriod int64
)

func applyJitter(t time.Duration) time.Duration {
	multiplier := rand.Float32() + 0.5
	return time.Duration(multiplier * float32(t))
}

func main() {
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

	ctx := context.Background()

	for {
		select {
		case sig := <-signalsC:
			log.Infof("Caught %s signal", sig)
			log.Info("Chaos monkey terminated")
			return
		case <-time.After(nextInterval):
			nextInterval = applyJitter(interval)

			podList, err := client.CoreV1().Pods("stackrox").List(ctx, v1.ListOptions{
				LabelSelector: "app=central",
			})
			if err != nil {
				log.Panicf("error listing pods: %v", err)
			}
			if len(podList.Items) == 0 {
				log.Info("No Central pods in this iteration. Will try again")
				continue
			}
			for _, pod := range podList.Items {
				log.Infof("Deleting pod %s", pod.Name)
				err := client.CoreV1().Pods("stackrox").Delete(ctx, pod.Name, v1.DeleteOptions{
					GracePeriodSeconds: &gracePeriod,
				})
				if err != nil {
					log.Errorf("error deleting pod %s: %v", pod.Name, err)
				}
			}
		}
	}
}
