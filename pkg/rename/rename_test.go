package rename

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- ConfigMap rename tests ---

func TestRenameConfigMap_Success(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old-config", Namespace: "default"},
		Data:       map[string]string{"key": "value"},
	}
	client := allowAll()
	client.Tracker().Add(cm)

	err := renameConfigMap(client, Options{
		OldName: "old-config", NewName: "new-config",
		Namespace: "default", Yes: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// new must exist
	newCM, err := client.CoreV1().ConfigMaps("default").Get(ctx(), "new-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("new configmap not found: %v", err)
	}
	if newCM.Data["key"] != "value" {
		t.Errorf("data not copied: got %v", newCM.Data)
	}

	// old must be gone
	_, err = client.CoreV1().ConfigMaps("default").Get(ctx(), "old-config", metav1.GetOptions{})
	if err == nil {
		t.Error("old configmap should have been deleted")
	}
}

func TestRenameConfigMap_NotFound(t *testing.T) {
	client := allowAll()
	err := renameConfigMap(client, Options{
		OldName: "missing", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error for missing configmap")
	}
}

func TestRenameConfigMap_NewNameAlreadyExists(t *testing.T) {
	old := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"}}
	existing := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"}}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(existing)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error when new name already exists")
	}
}

func TestRenameConfigMap_PreservesLabelsAndAnnotations(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "old",
			Namespace:   "default",
			Labels:      map[string]string{"app": "myapp"},
			Annotations: map[string]string{"note": "important"},
		},
		Data: map[string]string{"k": "v"},
	}
	client := allowAll()
	client.Tracker().Add(cm)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newCM, _ := client.CoreV1().ConfigMaps("default").Get(ctx(), "new", metav1.GetOptions{})
	if newCM.Labels["app"] != "myapp" {
		t.Errorf("labels not preserved: %v", newCM.Labels)
	}
	if newCM.Annotations["note"] != "important" {
		t.Errorf("annotations not preserved: %v", newCM.Annotations)
	}
}

func TestRenameConfigMap_DryRun(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
	}
	client := allowAll()
	client.Tracker().Add(cm)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", DryRun: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// old must still exist
	_, err = client.CoreV1().ConfigMaps("default").Get(ctx(), "old", metav1.GetOptions{})
	if err != nil {
		t.Error("dry-run should not delete old configmap")
	}

	// new must NOT exist
	_, err = client.CoreV1().ConfigMaps("default").Get(ctx(), "new", metav1.GetOptions{})
	if err == nil {
		t.Error("dry-run should not create new configmap")
	}
}

// --- Secret rename tests ---

func TestRenameSecret_Success(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old-secret", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"password": []byte("s3cr3t")},
	}
	client := allowAll()
	client.Tracker().Add(secret)

	err := renameSecret(client, Options{
		OldName: "old-secret", NewName: "new-secret",
		Namespace: "default", Yes: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSecret, err := client.CoreV1().Secrets("default").Get(ctx(), "new-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("new secret not found: %v", err)
	}
	if string(newSecret.Data["password"]) != "s3cr3t" {
		t.Errorf("secret data not copied correctly")
	}
	if newSecret.Type != corev1.SecretTypeOpaque {
		t.Errorf("secret type not preserved: got %v", newSecret.Type)
	}

	_, err = client.CoreV1().Secrets("default").Get(ctx(), "old-secret", metav1.GetOptions{})
	if err == nil {
		t.Error("old secret should have been deleted")
	}
}

func TestRenameSecret_NotFound(t *testing.T) {
	client := allowAll()
	err := renameSecret(client, Options{
		OldName: "missing", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
}

func TestRenameSecret_PreservesType(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
	}
	client := allowAll()
	client.Tracker().Add(secret)

	err := renameSecret(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newSecret, _ := client.CoreV1().Secrets("default").Get(ctx(), "new", metav1.GetOptions{})
	if newSecret.Type != corev1.SecretTypeTLS {
		t.Errorf("secret type not preserved: got %v", newSecret.Type)
	}
}

func TestRenameSecret_DryRun(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
	}
	client := allowAll()
	client.Tracker().Add(secret)

	err := renameSecret(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", DryRun: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = client.CoreV1().Secrets("default").Get(ctx(), "old", metav1.GetOptions{})
	if err != nil {
		t.Error("dry-run should not delete old secret")
	}

	_, err = client.CoreV1().Secrets("default").Get(ctx(), "new", metav1.GetOptions{})
	if err == nil {
		t.Error("dry-run should not create new secret")
	}
}
