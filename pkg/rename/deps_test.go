package rename

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func ctx() context.Context { return context.Background() }

// --- ConfigMap dependency tests ---

func TestFindConfigMapRefs_PodVolume(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name:         "cfg-vol",
				VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}},
			}},
			Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findConfigMapRefs(client, "default", "app-config")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindConfigMapRefs_PodEnvFrom(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx",
				EnvFrom: []corev1.EnvFromSource{{
					ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}},
				}},
			}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findConfigMapRefs(client, "default", "app-config")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindConfigMapRefs_PodEnvValueFrom(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx",
				Env: []corev1.EnvVar{{
					Name: "DB_HOST",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
							Key:                  "db_host",
						},
					},
				}},
			}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findConfigMapRefs(client, "default", "app-config")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindConfigMapRefs_DeploymentVolume(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploy", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name:         "cfg-vol",
						VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}},
					}},
					Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
				},
			},
		},
	}
	client := fake.NewSimpleClientset(dep)
	refs := findConfigMapRefs(client, "default", "app-config")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindConfigMapRefs_NoRefs(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "nginx"}}},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findConfigMapRefs(client, "default", "app-config")
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d: %v", len(refs), refs)
	}
}

func TestFindConfigMapRefs_DifferentNamespace(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "other"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name:         "cfg-vol",
				VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"}}},
			}},
			Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	// searching in "default" should not find pod in "other"
	refs := findConfigMapRefs(client, "default", "app-config")
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs across namespaces, got %d: %v", len(refs), refs)
	}
}

// --- Secret dependency tests ---

func TestFindSecretRefs_PodVolume(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name:         "sec-vol",
				VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "db-creds"}},
			}},
			Containers: []corev1.Container{{Name: "app", Image: "nginx"}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findSecretRefs(client, "default", "db-creds")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindSecretRefs_PodEnvFrom(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx",
				EnvFrom: []corev1.EnvFromSource{{
					SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "db-creds"}},
				}},
			}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findSecretRefs(client, "default", "db-creds")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindSecretRefs_ImagePullSecret(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry-creds"}},
			Containers:       []corev1.Container{{Name: "app", Image: "nginx"}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findSecretRefs(client, "default", "registry-creds")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindSecretRefs_DeploymentImagePullSecret(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-deploy", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: "registry-creds"}},
					Containers:       []corev1.Container{{Name: "app", Image: "nginx"}},
				},
			},
		},
	}
	client := fake.NewSimpleClientset(dep)
	refs := findSecretRefs(client, "default", "registry-creds")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}

func TestFindSecretRefs_NoRefs(t *testing.T) {
	client := fake.NewSimpleClientset()
	refs := findSecretRefs(client, "default", "db-creds")
	if len(refs) != 0 {
		t.Fatalf("expected 0 refs, got %d: %v", len(refs), refs)
	}
}

func TestFindSecretRefs_PodEnvValueFrom(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pod", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx",
				Env: []corev1.EnvVar{{
					Name: "DB_PASS",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "db-creds"},
							Key:                  "password",
						},
					},
				}},
			}},
		},
	}
	client := fake.NewSimpleClientset(pod)
	refs := findSecretRefs(client, "default", "db-creds")
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d: %v", len(refs), refs)
	}
}
