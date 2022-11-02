package subscription

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"context"
	"k8s.io/klog/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodSubscription struct {
	watcherInterface watch.Interface
	ClientSet kubernetes.Interface
	Ctx context.Context
        ConfigMapSubscriptRef *ConfigMapSubscription
}

func (p *PodSubscription) applyConfigMapChanges(pod *v1.Pod, event watch.EventType) {
        if p.ConfigMapSubscriptRef == nil || p.ConfigMapSubscriptRef.PlatformConfig == nil {
                return
        }

        updatedPod := pod.DeepCopy()
        if updatedPod.Annotations == nil {
                updatedPod.Annotations = make(map[string]string)
        }

        for _, annotation := range p.ConfigMapSubscriptRef.PlatformConfig.Annotations {
                updatedPod.Annotations[annotation.Name] = annotation.Value
        }

        _, err := p.ClientSet.CoreV1().Pods(pod.Namespace).Update(p.Ctx, updatedPod, metav1.UpdateOptions{})
        if err != nil {
                klog.Error(err)
        }
}

func (p *PodSubscription) Reconcile(object runtime.Object, event watch.EventType) {
	pod := object.(*v1.Pod)
	klog.Infof("PodSubscription eventType %s for %s", event, pod.Name)

	switch event {
	case watch.Added:
		p.applyConfigMapChanges(pod, event)
        case watch.Deleted:
	case watch.Modified:
                p.applyConfigMapChanges(pod, event)
	}
}


func (p *PodSubscription) Subscribe() (watch.Interface, error) {
	var err error
	p.watcherInterface, err = p.ClientSet.CoreV1().Pods("").Watch(p.Ctx, metav1.ListOptions{})
	if err != nil {
		klog.Fatalf("watch interface: %s", err.Error())
		return nil, err
	}

	return p.watcherInterface, nil
}

