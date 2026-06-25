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

func TestRenameConfigMap_NewNameAlreadyExistsWithDifferentData(t *testing.T) {
	// New name exists with DIFFERENT data — not a partial rename, must error.
	old := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Data:       map[string]string{"key": "old-value"},
	}
	existing := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Data:       map[string]string{"key": "different-value"},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(existing)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error when new name already exists with different data")
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

// --- Secret type preservation tests ---

func TestRenameSecret_PreservesDockerConfigJSON(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Type:       corev1.SecretTypeDockerConfigJson,
		Data:       map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`)},
	}
	client := allowAll()
	client.Tracker().Add(secret)

	if err := renameSecret(client, Options{OldName: "old", NewName: "new", Namespace: "default", Yes: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	newSecret, _ := client.CoreV1().Secrets("default").Get(ctx(), "new", metav1.GetOptions{})
	if newSecret.Type != corev1.SecretTypeDockerConfigJson {
		t.Errorf("type not preserved: got %v", newSecret.Type)
	}
	if string(newSecret.Data[corev1.DockerConfigJsonKey]) != `{"auths":{}}` {
		t.Errorf("data not preserved: got %v", newSecret.Data)
	}
}

func TestRenameSecret_PreservesServiceAccountToken(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Type:       corev1.SecretTypeServiceAccountToken,
		Data: map[string][]byte{
			"token":     []byte("mytoken"),
			"namespace": []byte("default"),
		},
	}
	client := allowAll()
	client.Tracker().Add(secret)

	if err := renameSecret(client, Options{OldName: "old", NewName: "new", Namespace: "default", Yes: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	newSecret, _ := client.CoreV1().Secrets("default").Get(ctx(), "new", metav1.GetOptions{})
	if newSecret.Type != corev1.SecretTypeServiceAccountToken {
		t.Errorf("type not preserved: got %v", newSecret.Type)
	}
}

// --- Partial-failure recovery tests ---

func TestRenameConfigMap_PartialRecovery_SameData(t *testing.T) {
	// Simulate: CREATE succeeded, DELETE never ran (crash/Ctrl+C).
	// Both old and new exist with identical data.
	old := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Data:       map[string]string{"key": "value"},
	}
	newAlready := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Data:       map[string]string{"key": "value"},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newAlready)

	err := renameConfigMap(client, Options{OldName: "old", NewName: "new", Namespace: "default", Yes: true})
	if err != nil {
		t.Fatalf("expected recovery to succeed, got: %v", err)
	}

	// old must be gone after recovery
	_, getErr := client.CoreV1().ConfigMaps("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr == nil {
		t.Error("old configmap should have been deleted during recovery")
	}
}

func TestRenameConfigMap_PartialRecovery_DifferentData(t *testing.T) {
	// Both names exist but with different data — tool cannot auto-resolve.
	old := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Data:       map[string]string{"key": "old-value"},
	}
	newConflict := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Data:       map[string]string{"key": "different-value"},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newConflict)

	err := renameConfigMap(client, Options{OldName: "old", NewName: "new", Namespace: "default", Yes: true})
	if err == nil {
		t.Fatal("expected error when both names exist with different data")
	}
}

func TestRenameSecret_PartialRecovery_SameData(t *testing.T) {
	old := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"pass": []byte("secret")},
	}
	newAlready := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"pass": []byte("secret")},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newAlready)

	err := renameSecret(client, Options{OldName: "old", NewName: "new", Namespace: "default", Yes: true})
	if err != nil {
		t.Fatalf("expected recovery to succeed, got: %v", err)
	}

	_, getErr := client.CoreV1().Secrets("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr == nil {
		t.Error("old secret should have been deleted during recovery")
	}
}

// dry-run on recovery path must not delete
func TestRenameConfigMap_Recovery_DryRun_DoesNotDelete(t *testing.T) {
	old := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Data:       map[string]string{"key": "value"},
	}
	newAlready := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Data:       map[string]string{"key": "value"},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newAlready)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", DryRun: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// old must still exist — dry-run must not complete the delete
	_, getErr := client.CoreV1().ConfigMaps("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr != nil {
		t.Error("dry-run recovery should not delete old configmap")
	}
}

func TestRenameSecret_Recovery_DryRun_DoesNotDelete(t *testing.T) {
	old := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"k": []byte("v")},
	}
	newAlready := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"k": []byte("v")},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newAlready)

	err := renameSecret(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", DryRun: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, getErr := client.CoreV1().Secrets("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr != nil {
		t.Error("dry-run recovery should not delete old secret")
	}
}

// old resource disappears between pre-flight and GET — must fail cleanly, nothing written
func TestRenameConfigMap_OldDisappearsBeforeCreate(t *testing.T) {
	// No old resource exists at all — simulates deletion mid-run
	client := allowAll()

	err := renameConfigMap(client, Options{
		OldName: "vanished", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error when old resource does not exist")
	}

	// new must not have been created
	_, getErr := client.CoreV1().ConfigMaps("default").Get(ctx(), "new", metav1.GetOptions{})
	if getErr == nil {
		t.Error("new configmap should not have been created when old was missing")
	}
}

func TestRenameSecret_OldDisappearsBeforeCreate(t *testing.T) {
	client := allowAll()

	err := renameSecret(client, Options{
		OldName: "vanished", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error when old secret does not exist")
	}

	_, getErr := client.CoreV1().Secrets("default").Get(ctx(), "new", metav1.GetOptions{})
	if getErr == nil {
		t.Error("new secret should not have been created when old was missing")
	}
}

// labels/annotations difference must NOT block recovery — only data matters
func TestRenameConfigMap_Recovery_IgnoresLabelDifference(t *testing.T) {
	old := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default", Labels: map[string]string{"env": "staging"}},
		Data:       map[string]string{"key": "value"},
	}
	// new has same data but a controller added a label after CREATE
	newAlready := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default", Labels: map[string]string{"app.kubernetes.io/managed-by": "helm"}},
		Data:       map[string]string{"key": "value"},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newAlready)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err != nil {
		t.Fatalf("label difference should not block recovery, got: %v", err)
	}

	_, getErr := client.CoreV1().ConfigMaps("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr == nil {
		t.Error("old configmap should have been deleted during recovery")
	}
}

func TestRenameSecret_PartialRecovery_DifferentType(t *testing.T) {
	// Same name but different type — definitely not a partial rename, must error.
	old := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
		Type:       corev1.SecretTypeOpaque,
		Data:       map[string][]byte{"key": []byte("val")},
	}
	newConflict := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: "default"},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{"key": []byte("val")},
	}
	client := allowAll()
	client.Tracker().Add(old)
	client.Tracker().Add(newConflict)

	err := renameSecret(client, Options{OldName: "old", NewName: "new", Namespace: "default", Yes: true})
	if err == nil {
		t.Fatal("expected error when types differ")
	}
}
