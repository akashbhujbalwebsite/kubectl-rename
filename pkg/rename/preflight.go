package rename

import (
	"context"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type permissionResult struct {
	verb    string
	allowed bool
}

// checkRenamePermissions verifies the caller has get+create+delete on the resource
// before touching anything. Returns an error listing all missing verbs at once.
func checkRenamePermissions(client kubernetes.Interface, resource, namespace string) error {
	verbs := []string{"get", "create", "delete"}
	results := make([]permissionResult, len(verbs))

	for i, verb := range verbs {
		sar := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{
					Namespace: namespace,
					Verb:      verb,
					Resource:  resource,
				},
			},
		}
		resp, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(
			context.Background(), sar, metav1.CreateOptions{},
		)
		if err != nil {
			return fmt.Errorf("could not check %q permission on %s: %w", verb, resource, err)
		}
		results[i] = permissionResult{verb: verb, allowed: resp.Status.Allowed}
	}

	var missing []string
	for _, r := range results {
		if !r.allowed {
			missing = append(missing, fmt.Sprintf("'%s'", r.verb))
		}
	}

	if len(missing) == 0 {
		return nil
	}

	// Build a helpful error explaining exactly what's missing and why it matters.
	have := []string{}
	for _, r := range results {
		if r.allowed {
			have = append(have, fmt.Sprintf("'%s'", r.verb))
		}
	}

	msg := fmt.Sprintf("✗ Cannot rename: missing %s permission on %s in namespace %q",
		strings.Join(missing, " and "), resource, namespace)
	if len(have) > 0 {
		msg += fmt.Sprintf("\n  (you have %s but not %s — partial rename would leave duplicate resources)",
			strings.Join(have, "+"), strings.Join(missing, "+"))
	}
	return fmt.Errorf("%s", msg)
}
