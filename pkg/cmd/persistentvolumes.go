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

type PersistentVolumesOptions struct {
	IssuesOptions
}

func newPersistentVolumesOptions(options IssuesOptions) *PersistentVolumesOptions {
	return &PersistentVolumesOptions{
		IssuesOptions: options,
	}
}

func newPersistentVolumesCommand(factory cmdutil.Factory, options IssuesOptions) *cobra.Command {
	o := newPersistentVolumesOptions(options)

	cmd := &cobra.Command{
		Use:          "persistentvolumes",
		Aliases:      []string{"persistentvolume", "pv"},
		Short:        "List issues with PersistentVolumes",
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

func (o *PersistentVolumesOptions) Run(ctx context.Context, noHeader bool) error {
	client, err := o.GetClient()
	if err != nil {
		return err
	}

	pvs, err := client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	var matrix [][]string

	for _, pv := range pvs.Items {
		if pv.Status.Phase != "Bound" {
			accessModes := storageutil.GetAccessModesAsString(pv.Spec.AccessModes)
			storage := pv.Spec.Capacity[corev1.ResourceStorage]
			capacity := storage.String()

			row := []string{pv.Name, capacity, accessModes, string(pv.Spec.PersistentVolumeReclaimPolicy), string(pv.Status.Phase), fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name), pv.Spec.StorageClassName, pv.Status.Reason, utils.GetAge(pv.CreationTimestamp)}
			matrix = append(matrix, row)
		}
	}

	headers := []string{"NAME", "CAPACITY", "ACCESS MODES", "RECLAIM POLICY", "STATUS", "CLAIM", "STORAGECLASS", "REASON", "AGE"}

	buf := bytes.NewBuffer(nil)
	writer.WriteResults(buf, headers, matrix, noHeader)
	fmt.Printf("%s", buf.String())

	return nil
}
