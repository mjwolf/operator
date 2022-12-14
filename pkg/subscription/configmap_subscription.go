package subscription

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"context"
        "errors"
	"k8s.io/klog/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "gopkg.in/yaml.v2"
)

type ConfigMapSubscription struct {
	watcherInterface watch.Interface
	ClientSet kubernetes.Interface
	Ctx context.Context
        PlatformConfig *platformConfig
        PlatformConfigPhase string
	PodSubscriptRef *PodSubscription
}

var (
        platformConfigMapName = "platform-default-configmap"
        platformConfigMapNamespace = "kube-system"
)

type platformAnnotation struct {
        Name  string `yaml:"name"`
        Value string `yaml:"value"`
}

type platformConfig struct {
        Annotations []platformAnnotation `yaml:"annotations"`
}

func isPlatformConfigMap(configMap *v1.ConfigMap) (bool, error) {
        if configMap == nil {
                return false, errors.New("empty")
        }
        if configMap.Name == platformConfigMapName {
                return true, nil
        }
        return false, nil
}


func (p *ConfigMapSubscription) configChange(configMap *v1.ConfigMap, event watch.EventType) {
	p.PlatformConfigPhase = string(event)
	rawDefaultsString := configMap.Data["platform-defaults"]
	var unMarshalledData platformConfig
	err := yaml.Unmarshal([]byte(rawDefaultsString), &unMarshalledData)
	if err != nil {
		klog.Error(err)
		return
	}
	p.PlatformConfig = &unMarshalledData

	p.PodSubscriptRef.UpdatePods()
}

func (p *ConfigMapSubscription) Reconcile(object runtime.Object, event watch.EventType) {
	configMap := object.(*v1.ConfigMap)
	klog.Infof("ConfigMapSubscription eventType %s for %s", event, configMap.Name)

        if ok, _ := isPlatformConfigMap(configMap); !ok {
                klog.Info("Not a Platform ConfigMap: %s", configMap.Name)
                return
        }
	switch event {
	case watch.Added:
		p.configChange(configMap, event)
	case watch.Deleted:
                p.PlatformConfigPhase = string(event)
                p.PlatformConfig = nil
	case watch.Modified:
		p.configChange(configMap, event)
	}
}


func (p *ConfigMapSubscription) Subscribe() (watch.Interface, error) {
	var err error
	p.watcherInterface, err = p.ClientSet.CoreV1().ConfigMaps("").Watch(p.Ctx, metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("watch interface: %s", err.Error())
		return nil, err
	}

	return p.watcherInterface, nil
}

