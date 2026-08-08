package cmd

import (
	"context"
	"fmt"

	"github.com/ricoberger/kubectl-issues/pkg/client"
	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	storageutil "k8s.io/kubectl/pkg/util/storage"
)

type PersistentVolumeClaimsOptions struct {
	IssuesOptions
}

func newPersistentVolumeClaimsOptions(options IssuesOptions) *PersistentVolumeClaimsOptions {
	return &PersistentVolumeClaimsOptions{
		IssuesOptions: options,
	}
}

func newPersistentVolumeClaimsCommand(_ cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newPersistentVolumeClaimsOptions(options)

	cmd := &cobra.Command{
		Use:          "persistentvolumeclaims",
		Aliases:      []string{"persistentvolumeclaim", "pvc"},
		Short:        "List issues with PersistentVolumeClaims",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			headers := []string{"NAMESPACE", "NAME", "STATUS", "VOLUME", "CAPACITY", "ACCESS MODES", "STORAGECLASS", "AGE"}
			if err := o.RunAcrossContexts(ctx, noHeader, headers, o.rows); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.output, "output", "o", "", "Output format. One of: json.")

	return cmd
}

func (o *PersistentVolumeClaimsOptions) rows(ctx context.Context, contextClient client.ContextClient) ([][]string, error) {
	pvcs, err := contextClient.Client.CoreV1().PersistentVolumeClaims(contextClient.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var matrix [][]string

	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase != "Bound" {
			accessModes := storageutil.GetAccessModesAsString(pvc.Status.AccessModes)
			storage := pvc.Status.Capacity[corev1.ResourceStorage]
			capacity := storage.String()

			row := []string{pvc.Namespace, pvc.Name, string(pvc.Status.Phase), pvc.Spec.VolumeName, capacity, accessModes, *pvc.Spec.StorageClassName, utils.GetAge(pvc.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	return matrix, nil
}
