package cmd

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ricoberger/kubectl-issues/pkg/cmd/utils"
	"github.com/ricoberger/kubectl-issues/pkg/writer"

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

func newPersistentVolumeClaimsCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newPersistentVolumeClaimsOptions(options)

	cmd := &cobra.Command{
		Use:          "persistentvolumeclaims",
		Aliases:      []string{"persistentvolumeclaim", "pvc"},
		Short:        "List issues with PersistentVolumeClaims",
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(factory, c); err != nil {
				return err
			}

			ctx := context.Background()
			noHeader := c.Flag("no-headers").Changed
			if err := o.Run(ctx, noHeader); err != nil {
				fmt.Fprintln(options.Streams.ErrOut, err.Error())
				return nil
			}
			return nil
		},
	}

	o.ResourceBuilderFlags.AddFlags(cmd.Flags())

	return cmd
}

func (o *PersistentVolumeClaimsOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	pvcs, err := client.CoreV1().PersistentVolumeClaims(o.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
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

	headers := []string{"NAMESPACE", "NAME", "STATUS", "VOLUME", "CAPACITY", "ACCESS MODES", "STORAGECLASS", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
