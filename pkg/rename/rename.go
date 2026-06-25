package rename

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Options struct {
	Kind       string
	OldName    string
	NewName    string
	Namespace  string
	Kubeconfig string
	DryRun     bool
	Yes        bool
}

func Run(opts Options) error {
	client, err := buildClient(opts.Kubeconfig)
	if err != nil {
		return fmt.Errorf("cannot connect to cluster: %w", err)
	}

	switch opts.Kind {
	case "configmap":
		return renameConfigMap(client, opts)
	case "secret":
		return renameSecret(client, opts)
	default:
		return fmt.Errorf("unsupported kind: %s", opts.Kind)
	}
}

func renameConfigMap(client kubernetes.Interface, opts Options) error {
	ctx := context.Background()

	if err := checkRenamePermissions(client, "configmaps", opts.Namespace); err != nil {
		return err
	}

	cm, err := client.CoreV1().ConfigMaps(opts.Namespace).Get(ctx, opts.OldName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("configmap %q not found in namespace %q: %w", opts.OldName, opts.Namespace, err)
	}

	refs := findConfigMapRefs(client, opts.Namespace, opts.OldName)

	printPlan("ConfigMap", opts, refs)

	if opts.DryRun {
		fmt.Println("\n[dry-run] No changes made.")
		return nil
	}

	if !opts.Yes {
		if !confirm(fmt.Sprintf("Rename ConfigMap %q to %q in namespace %q?", opts.OldName, opts.NewName, opts.Namespace)) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	newCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        opts.NewName,
			Namespace:   opts.Namespace,
			Labels:      cm.Labels,
			Annotations: cm.Annotations,
		},
		Data:       cm.Data,
		BinaryData: cm.BinaryData,
	}

	if _, err := client.CoreV1().ConfigMaps(opts.Namespace).Create(ctx, newCM, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create new configmap %q: %w", opts.NewName, err)
	}
	fmt.Printf("✓ Created ConfigMap %q\n", opts.NewName)

	if err := client.CoreV1().ConfigMaps(opts.Namespace).Delete(ctx, opts.OldName, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("created %q but failed to delete old configmap %q: %w", opts.NewName, opts.OldName, err)
	}
	fmt.Printf("✓ Deleted ConfigMap %q\n", opts.OldName)
	fmt.Printf("\nDone. ConfigMap renamed: %q → %q\n", opts.OldName, opts.NewName)

	if len(refs) > 0 {
		fmt.Println("\n⚠️  Update the following resources to reference the new name:")
		for _, r := range refs {
			fmt.Printf("   %s\n", r)
		}
	}

	return nil
}

func renameSecret(client kubernetes.Interface, opts Options) error {
	ctx := context.Background()

	if err := checkRenamePermissions(client, "secrets", opts.Namespace); err != nil {
		return err
	}

	secret, err := client.CoreV1().Secrets(opts.Namespace).Get(ctx, opts.OldName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("secret %q not found in namespace %q: %w", opts.OldName, opts.Namespace, err)
	}

	refs := findSecretRefs(client, opts.Namespace, opts.OldName)

	printPlan("Secret", opts, refs)

	if opts.DryRun {
		fmt.Println("\n[dry-run] No changes made.")
		return nil
	}

	if !opts.Yes {
		if !confirm(fmt.Sprintf("Rename Secret %q to %q in namespace %q?", opts.OldName, opts.NewName, opts.Namespace)) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        opts.NewName,
			Namespace:   opts.Namespace,
			Labels:      secret.Labels,
			Annotations: secret.Annotations,
		},
		Type: secret.Type,
		Data: secret.Data,
	}

	if _, err := client.CoreV1().Secrets(opts.Namespace).Create(ctx, newSecret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to create new secret %q: %w", opts.NewName, err)
	}
	fmt.Printf("✓ Created Secret %q\n", opts.NewName)

	if err := client.CoreV1().Secrets(opts.Namespace).Delete(ctx, opts.OldName, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("created %q but failed to delete old secret %q: %w", opts.NewName, opts.OldName, err)
	}
	fmt.Printf("✓ Deleted Secret %q\n", opts.OldName)
	fmt.Printf("\nDone. Secret renamed: %q → %q\n", opts.OldName, opts.NewName)

	if len(refs) > 0 {
		fmt.Println("\n⚠️  Update the following resources to reference the new name:")
		for _, r := range refs {
			fmt.Printf("   %s\n", r)
		}
	}

	return nil
}

func printPlan(kind string, opts Options, refs []string) {
	fmt.Printf("\n %s Rename Plan\n", kind)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf(" Namespace : %s\n", opts.Namespace)
	fmt.Printf(" Old name  : %s\n", opts.OldName)
	fmt.Printf(" New name  : %s\n", opts.NewName)
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf(" Steps:\n")
	fmt.Printf("   1. GET    %s/%s\n", strings.ToLower(kind), opts.OldName)
	fmt.Printf("   2. CREATE %s/%s (same data)\n", strings.ToLower(kind), opts.NewName)
	fmt.Printf("   3. DELETE %s/%s\n", strings.ToLower(kind), opts.OldName)

	if len(refs) > 0 {
		fmt.Println(strings.Repeat("─", 50))
		fmt.Printf(" ⚠️  References found (manual update required after rename):\n")
		for _, r := range refs {
			fmt.Printf("   %s\n", r)
		}
	} else {
		fmt.Println(strings.Repeat("─", 50))
		fmt.Println(" No references found in this namespace.")
	}
	fmt.Println(strings.Repeat("─", 50))
}

func confirm(prompt string) bool {
	fmt.Printf("%s (yes/no): ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(strings.ToLower(scanner.Text())) == "yes"
}

func buildClient(kubeconfig string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}
