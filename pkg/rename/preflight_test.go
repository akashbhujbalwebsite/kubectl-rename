package rename

import (
	"strings"
	"testing"

	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// allowAll returns a fake client where every SelfSubjectAccessReview is allowed.
func allowAll() *fake.Clientset {
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: true}}, nil
	})
	return client
}

// denyVerbs returns a fake client that denies the specified verbs and allows everything else.
func denyVerbs(denied ...string) *fake.Clientset {
	deniedSet := map[string]bool{}
	for _, v := range denied {
		deniedSet[v] = true
	}
	client := fake.NewSimpleClientset()
	client.PrependReactor("create", "selfsubjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		obj := action.(k8stesting.CreateAction).GetObject().(*authv1.SelfSubjectAccessReview)
		verb := obj.Spec.ResourceAttributes.Verb
		allowed := !deniedSet[verb]
		return true, &authv1.SelfSubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
	})
	return client
}

func TestCheckRenamePermissions_AllAllowed(t *testing.T) {
	err := checkRenamePermissions(allowAll(), "configmaps", "default")
	if err != nil {
		t.Fatalf("expected no error when all permissions allowed, got: %v", err)
	}
}

func TestCheckRenamePermissions_MissingDelete(t *testing.T) {
	err := checkRenamePermissions(denyVerbs("delete"), "configmaps", "default")
	if err == nil {
		t.Fatal("expected error when delete is denied")
	}
	if !strings.Contains(err.Error(), "'delete'") {
		t.Errorf("error should mention missing verb 'delete', got: %v", err)
	}
	if !strings.Contains(err.Error(), "partial rename") {
		t.Errorf("error should mention partial rename risk, got: %v", err)
	}
}

func TestCheckRenamePermissions_MissingCreate(t *testing.T) {
	err := checkRenamePermissions(denyVerbs("create"), "configmaps", "default")
	if err == nil {
		t.Fatal("expected error when create is denied")
	}
	if !strings.Contains(err.Error(), "'create'") {
		t.Errorf("error should mention missing verb 'create', got: %v", err)
	}
}

func TestCheckRenamePermissions_MissingGet(t *testing.T) {
	err := checkRenamePermissions(denyVerbs("get"), "configmaps", "default")
	if err == nil {
		t.Fatal("expected error when get is denied")
	}
	if !strings.Contains(err.Error(), "'get'") {
		t.Errorf("error should mention missing verb 'get', got: %v", err)
	}
}

func TestCheckRenamePermissions_MissingMultiple(t *testing.T) {
	err := checkRenamePermissions(denyVerbs("create", "delete"), "configmaps", "default")
	if err == nil {
		t.Fatal("expected error when create and delete are denied")
	}
	if !strings.Contains(err.Error(), "'create'") || !strings.Contains(err.Error(), "'delete'") {
		t.Errorf("error should mention both missing verbs, got: %v", err)
	}
}

func TestCheckRenamePermissions_MissingAll(t *testing.T) {
	err := checkRenamePermissions(denyVerbs("get", "create", "delete"), "configmaps", "default")
	if err == nil {
		t.Fatal("expected error when all permissions denied")
	}
}

func TestRenameConfigMap_FailsOnMissingPermission(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
	}
	client := denyVerbs("delete")
	client.Tracker().Add(cm)

	err := renameConfigMap(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error when delete permission is missing")
	}

	// Old configmap must still exist — nothing should have been touched
	_, getErr := client.CoreV1().ConfigMaps("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr != nil {
		t.Error("old configmap was deleted despite missing permission — pre-flight check failed")
	}
}

func TestRenameSecret_FailsOnMissingPermission(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: "default"},
	}
	client := denyVerbs("create")
	client.Tracker().Add(secret)

	err := renameSecret(client, Options{
		OldName: "old", NewName: "new", Namespace: "default", Yes: true,
	})
	if err == nil {
		t.Fatal("expected error when create permission is missing")
	}

	// Old secret must still exist
	_, getErr := client.CoreV1().Secrets("default").Get(ctx(), "old", metav1.GetOptions{})
	if getErr != nil {
		t.Error("old secret was deleted despite missing permission")
	}
}
