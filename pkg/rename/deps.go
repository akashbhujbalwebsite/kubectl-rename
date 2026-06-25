package rename

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// findConfigMapRefs returns a list of resources in the namespace that reference the named ConfigMap.
func findConfigMapRefs(client kubernetes.Interface, namespace, name string) []string {
	ctx := context.Background()
	var refs []string

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			for _, vol := range pod.Spec.Volumes {
				if vol.ConfigMap != nil && vol.ConfigMap.Name == name {
					refs = append(refs, fmt.Sprintf("Pod/%s (volume: %s)", pod.Name, vol.Name))
				}
			}
			for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				for _, env := range c.EnvFrom {
					if env.ConfigMapRef != nil && env.ConfigMapRef.Name == name {
						refs = append(refs, fmt.Sprintf("Pod/%s (envFrom in container %s)", pod.Name, c.Name))
					}
				}
				for _, env := range c.Env {
					if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == name {
						refs = append(refs, fmt.Sprintf("Pod/%s (env %s in container %s)", pod.Name, env.Name, c.Name))
					}
				}
			}
		}
	}

	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, d := range deployments.Items {
			for _, vol := range d.Spec.Template.Spec.Volumes {
				if vol.ConfigMap != nil && vol.ConfigMap.Name == name {
					refs = append(refs, fmt.Sprintf("Deployment/%s (volume: %s)", d.Name, vol.Name))
				}
			}
			for _, c := range append(d.Spec.Template.Spec.Containers, d.Spec.Template.Spec.InitContainers...) {
				for _, env := range c.EnvFrom {
					if env.ConfigMapRef != nil && env.ConfigMapRef.Name == name {
						refs = append(refs, fmt.Sprintf("Deployment/%s (envFrom in container %s)", d.Name, c.Name))
					}
				}
			}
		}
	}

	return refs
}

// findSecretRefs returns a list of resources in the namespace that reference the named Secret.
func findSecretRefs(client kubernetes.Interface, namespace, name string) []string {
	ctx := context.Background()
	var refs []string

	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			for _, vol := range pod.Spec.Volumes {
				if vol.Secret != nil && vol.Secret.SecretName == name {
					refs = append(refs, fmt.Sprintf("Pod/%s (volume: %s)", pod.Name, vol.Name))
				}
			}
			for _, c := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				for _, env := range c.EnvFrom {
					if env.SecretRef != nil && env.SecretRef.Name == name {
						refs = append(refs, fmt.Sprintf("Pod/%s (envFrom in container %s)", pod.Name, c.Name))
					}
				}
				for _, env := range c.Env {
					if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == name {
						refs = append(refs, fmt.Sprintf("Pod/%s (env %s in container %s)", pod.Name, env.Name, c.Name))
					}
				}
			}
			for _, s := range pod.Spec.ImagePullSecrets {
				if s.Name == name {
					refs = append(refs, fmt.Sprintf("Pod/%s (imagePullSecret)", pod.Name))
				}
			}
		}
	}

	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, d := range deployments.Items {
			for _, vol := range d.Spec.Template.Spec.Volumes {
				if vol.Secret != nil && vol.Secret.SecretName == name {
					refs = append(refs, fmt.Sprintf("Deployment/%s (volume: %s)", d.Name, vol.Name))
				}
			}
			for _, c := range append(d.Spec.Template.Spec.Containers, d.Spec.Template.Spec.InitContainers...) {
				for _, env := range c.EnvFrom {
					if env.SecretRef != nil && env.SecretRef.Name == name {
						refs = append(refs, fmt.Sprintf("Deployment/%s (envFrom in container %s)", d.Name, c.Name))
					}
				}
			}
			for _, s := range d.Spec.Template.Spec.ImagePullSecrets {
				if s.Name == name {
					refs = append(refs, fmt.Sprintf("Deployment/%s (imagePullSecret)", d.Name))
				}
			}
		}
	}

	return refs
}
