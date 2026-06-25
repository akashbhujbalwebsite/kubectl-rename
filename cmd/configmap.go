package cmd

import (
	"github.com/akashbhujbalwebsite/kubectl-rename/pkg/rename"
	"github.com/spf13/cobra"
)

func newConfigMapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configmap OLD_NAME NEW_NAME",
		Short: "Rename a ConfigMap",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rename.Run(rename.Options{
				Kind:       "configmap",
				OldName:    args[0],
				NewName:    args[1],
				Namespace:  namespace,
				Kubeconfig: kubeconfig,
				DryRun:     dryRun,
				Yes:        yes,
			})
		},
	}
}
