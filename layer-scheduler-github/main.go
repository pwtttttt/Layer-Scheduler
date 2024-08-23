package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"layer-scheduler/layer"

	klog "k8s.io/klog/v2"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func main() {
	waitTs := 10 * time.Second
	localCacheFile := "/etc/kubernetes/cache.json"
	klog.Infof("启动监听器")
	reg, err := layer.NewRegistry(
		"Your Docker Registry Site",
		"Account",
		"Password",
	)
	if err != nil {
		klog.Fatalf("监听器启动失败, err: %s", err)
		os.Exit(2)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go reg.Watcher(waitTs, localCacheFile, ctx)

	klog.Infof("启动调度器")
	command := app.NewSchedulerCommand(
		app.WithPlugin(layer.Name, layer.New),
	)
	// time.Sleep(100 * time.Second)
	defer cancel()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
