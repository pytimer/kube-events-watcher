package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/pytimer/kube-events-watcher/constants"
	"github.com/pytimer/kube-events-watcher/flags"
	"github.com/pytimer/kube-events-watcher/sinks"
	"github.com/pytimer/kube-events-watcher/watchers"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
	"k8s.io/klog"
)

var (
	level        int
	kubeconfig   string
	resyncPeriod time.Duration
	sink         flags.Uri
)

func listenSystemStopSignal() chan struct{} {
	ch := make(chan struct{})
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGINT, syscall.SIGTERM)
		sig := <-c
		klog.Infof("Received signal %s, terminating\n", sig.String())
		close(ch)
	}()
	return ch
}

func newKubernetesClient() (kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create out-cluster config: %v", err)
		}
	}

	// Use protobufs for communication with apiserver.
	config.ContentType = "application/vnd.kubernetes.protobuf"

	return kubernetes.NewForConfig(config)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	logs.InitLogs()
	defer logs.FlushLogs()

	pflag.IntVarP(&level, "v", "v", 0, "log level for V logs")
	pflag.StringVar(&kubeconfig, "kubeconfig", "", `absolute path to the kubeconfig file. 
Optional, if the kubeconfig empty, this controller is running in a kubernetes cluster.`)
	pflag.DurationVar(&resyncPeriod, "resync-period", constants.DefaultResyncPeriod, "Watcher reflector resync period")
	pflag.Var(&sink, "sink", `Sink type to save the kubernetes events. e.g. --sink=elasticsearch:http://elasticsearch.com:9200`)
	pflag.Parse()

	logs.GlogSetter(strconv.Itoa(level))

	klog.Info("kube-events-watcher starting...")
	stopCh := listenSystemStopSignal()

	outSink, err := sinks.NewEventSinkManager(sink)
	if err != nil {
		klog.Fatalf("Failed to initialize sink: %v", err)
	}

	client, err := newKubernetesClient()
	if err != nil {
		klog.Fatalf("Failed to initialize kubernetes client: %v", err)
	}

	eventWatcher := watchers.NewEventWatcher(client, outSink, resyncPeriod)
	eventWatcher.Run(stopCh)
}
