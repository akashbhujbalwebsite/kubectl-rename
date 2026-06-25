package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
	namespace  string
	dryRun     bool
	yes        bool
)

var rootCmd = &cobra.Command{
	Use:   "kubectl-rename",
	Short: "Safely rename Kubernetes resources",
	Long:  "Rename ConfigMaps and Secrets by creating a copy with the new name and deleting the old one.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", clientcmd.RecommendedHomeFile, "path to kubeconfig")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "namespace of the resource")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would happen without making changes")
	rootCmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompt")

	rootCmd.AddCommand(newConfigMapCmd())
	rootCmd.AddCommand(newSecretCmd())
}
