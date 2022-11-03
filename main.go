

package main

import (
	"context"
	"flag"
	"k8s.io/klog/v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/mjwolf/operator/pkg/subscription"
	"github.com/mjwolf/operator/pkg/runtime"
)

var (
	addr = flag.String("listen-addr", ":8080", "the listen address")
	kubeconfig string
	masterURL string
)

func main() {
	flag.Parse()

	klog.Info("Starting operator...")

	kubeCfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	defaultKubernetesClientSet, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		klog.Fatalf("watcher clientset: %s", err.Error())
	}

	// Context
	context := context.TODO()

        configMapSubscription := &subscription.ConfigMapSubscription{
                ClientSet: defaultKubernetesClientSet,
                Ctx: context,
	}

        podSubscription := &subscription.PodSubscription{
	        ClientSet: defaultKubernetesClientSet,
		Ctx: context,
                ConfigMapSubscriptRef: configMapSubscription,
        }

	configMapSubscription.PodSubscriptRef = podSubscription

        if err := runtime.RunLoop([]subscription.ISubscription{
	        podSubscription,
                configMapSubscription,
        }); err != nil {
		klog.Fatalf(err.Error())
	}

	klog.Info("End of main!")
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	flag.StringVar(&masterURL, "master", "", "The address of the k8s server")
}

