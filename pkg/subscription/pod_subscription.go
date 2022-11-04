package subscription

import (
	"errors"
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
	knownPods []*v1.Pod
}

func (p *PodSubscription) applyConfigMapChanges(pod *v1.Pod) *v1.Pod {
        if p.ConfigMapSubscriptRef == nil || p.ConfigMapSubscriptRef.PlatformConfig == nil {
                return nil
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
	return updatedPod
}

func (p *PodSubscription) UpdatePods() error {
	if p.knownPods == nil {
		return errors.New("knownPods not initialized!")
	}
	for _, pod := range p.knownPods {
		if pod != nil {
			p.applyConfigMapChanges(pod)
		}
	}
	return nil
}

func (p *PodSubscription) maybeDeletePod(pod *v1.Pod) (bool, error) {
	if pod.Annotations != nil && pod.Annotations["action"] == "delete" {
		p.ClientSet.CoreV1().Pods(pod.Namespace).Delete(p.Ctx, pod.Name, metav1.DeleteOptions{})

		return true, nil
	}
	return false, nil
}

func (p *PodSubscription) Reconcile(object runtime.Object, event watch.EventType) {
	pod := object.(*v1.Pod)
	klog.Infof("PodSubscription eventType %s for %s", event, pod.Name)

	switch event {
	case watch.Added:
		pod = p.applyConfigMapChanges(pod)
		if p.knownPods == nil {
			p.knownPods = make([]*v1.Pod, 1)
		}
		p.knownPods = append(p.knownPods, pod)
        case watch.Deleted:
	case watch.Modified:
		if deleted, err := p.maybeDeletePod(pod); deleted || err != nil {
			return
		}
		pod = p.applyConfigMapChanges(pod)
		if p.knownPods == nil {
			p.knownPods = make([]*v1.Pod, 1)
		}
		p.knownPods = append(p.knownPods, pod)
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

