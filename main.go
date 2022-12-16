package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd := rootCmd()
	cmd.AddCommand(listCmd())
	cmd.AddCommand(nodesCmd())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := cmd.ExecuteContext(ctx)
	cancel()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	kflags := genericclioptions.NewConfigFlags(true)

	cmd := &cobra.Command{
		Use:   "kubectl nodepools",
		Short: "Read-only interaction with nodepools",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cfg, err := kflags.ToRESTConfig()
			if err != nil {
				return err
			}

			klient, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return err
			}

			ctx = context.WithValue(ctx, "klient", klient)

			cmd.SetContext(ctx)

			return nil
		},
	}

	flags := cmd.PersistentFlags()
	kflags.AddFlags(flags)

	return cmd
}

var providerNodepoolLabels = []string{
	"eks.amazonaws.com/nodegroup",   // AWS
	"cloud.google.com/gke-nodepool", // GCP
	"agentpool",                     // AKS
}

func findNodepool(node corev1.Node) string {
	for _, lbl := range providerNodepoolLabels {
		if np, ok := node.Labels[lbl]; ok {
			return np
		}
	}

	return ""
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			klient := ctx.Value("klient").(*kubernetes.Clientset)

			res, err := klient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			nps := make(map[string]uint)
			names := make([]string, 0, len(nps))

			for _, n := range res.Items {
				np := findNodepool(n)
				nps[np] += 1
				if nps[np] == 1 {
					names = append(names, np)
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)

			fmt.Fprintln(w, "NAME\tNODES")

			sort.Strings(names)
			for _, n := range names {
				fmt.Fprintf(w, "%s\t%5d\n", n, nps[n])
			}

			return w.Flush()
		},
	}

	return cmd
}

func nodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("need to pass a single nodepool name")
			}

			ctx := cmd.Context()

			klient := ctx.Value("klient").(*kubernetes.Clientset)

			res, err := klient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			var ns []string

			for _, n := range res.Items {
				if np := findNodepool(n); np == args[0] {
					ns = append(ns, n.Name)
				}
			}

			sort.Strings(ns)
			for _, n := range ns {
				fmt.Println(n)
			}

			return nil
		},
	}

	return cmd
}
