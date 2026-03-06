// Package main implements the example.
package main

import (
	"log"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/udhos/kube-informer-serviceaccount/serviceaccountinformer"
	"github.com/udhos/kube/kubeclient"
)

func main() {

	//
	// debug
	//
	debugStr := os.Getenv("DEBUG")
	var debug bool
	if debugStr != "" {
		var errConv error
		debug, errConv = strconv.ParseBool(debugStr)
		if errConv != nil {
			slog.Error("main env", "DEBUG", debugStr, "error", errConv)
		}
		if debug {
			handlerOptions := &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}
			logger := slog.New(slog.NewTextHandler(os.Stderr, handlerOptions))
			slog.SetDefault(logger)
		}
	}
	slog.Info("main env", "DEBUG", debugStr, "debug", debug)

	//
	// interval
	//
	intervalStr := os.Getenv("INTERVAL")
	interval, errConv := time.ParseDuration(intervalStr)
	if errConv != nil {
		log.Printf("INTERVAL='%s': %v", intervalStr, errConv)
		interval = 10 * time.Minute
	}
	log.Printf("INTERVAL='%s' interval=%v", intervalStr, interval)

	//
	// label selector
	//
	labelSelector := ""
	ls := os.Getenv("LABEL_SELECTOR")
	if ls != "" {
		labelSelector = ls
	}
	log.Printf("LABEL_SELECTOR='%s' label_selector=%s", ls, labelSelector)

	//
	// namespace
	//
	namespace := ""
	ns := os.Getenv("NAMESPACE")
	if ns != "" {
		namespace = ns
	}
	log.Printf("NAMESPACE='%s' namespace=%s", ns, namespace)

	//
	// resync period
	//
	resyncStr := os.Getenv("RESYNC_PERIOD")
	resync, errSync := time.ParseDuration(resyncStr)
	if errSync != nil {
		log.Printf("RESYNC_PERIOD='%s': %v", resyncStr, errSync)
	}
	log.Printf("RESYNC_PERIOD='%s' resync_period=%v", resyncStr, resync)

	//
	// kube client
	//
	clientOptions := kubeclient.Options{DebugLog: true}
	clientset, errClientset := kubeclient.New(clientOptions)
	if errClientset != nil {
		log.Fatalf("kube clientset error: %v", errClientset)
	}

	options := serviceaccountinformer.Options{
		Client:        clientset,
		Namespace:     namespace,
		LabelSelector: labelSelector,
		OnUpdate:      onUpdate,
		ResyncPeriod:  resync,
	}

	informer := serviceaccountinformer.New(options)

	go func() {
		log.Printf("######## main: time limit: %v - begin", interval)
		time.Sleep(interval)
		log.Printf("######## main: time limit: %v - end", interval)
		informer.Stop()
	}()

	errRun := informer.Run()
	log.Printf("informer run error: %v", errRun)
}

var update int

func onUpdate(serviceAccounts []serviceaccountinformer.ServiceAccount) {
	const me = "onUpdate"
	update++
	log.Printf("%s: update=%d: service accounts: %d",
		me, update, len(serviceAccounts))
	for i, sa := range serviceAccounts {
		log.Printf("%s: update=%d %d/%d: namespace=%s serviceAccount=%s annotations=%v",
			me, update, i, len(serviceAccounts), sa.Name, sa.Namespace, sa.Annotations)
	}
}
