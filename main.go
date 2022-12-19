package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
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

	// make list the default command
	if c, _, err := cmd.Find(os.Args[1:]); err == nil && c.Use == cmd.Use {
		cmd.SetArgs(append([]string{"list"}, os.Args[1:]...))
	}

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

var providerNodepoolLabels = map[string]string{
	"AWS":  "eks.amazonaws.com/nodegroup",
	"GCP":  "cloud.google.com/gke-nodepool",
	"AKS":  "kubernetes.azure.com/agentpool",
	"DOKS": "doks.digitalocean.com/node-pool-id",
}

func findNodepool(node corev1.Node) string {
	for _, lbl := range providerNodepoolLabels {
		if np, ok := node.Labels[lbl]; ok {
			return np
		}
	}

	return "-"
}

func instanceType(node corev1.Node) string {
	t, ok := node.Labels["node.kubernetes.io/instance-type"]
	if ok {
		return t
	}
	t, ok = node.Labels["beta.kubernetes.io/instance-type"]
	if ok {
		return t
	}
	return "-"
}

type nodepool struct {
	Name  string
	Type  string
	Nodes uint
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			klient := ctx.Value("klient").(kubernetes.Interface)

			res, err := klient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			nps := make(map[string]*nodepool)
			names := make([]string, 0, len(nps))

			for _, n := range res.Items {
				npName := findNodepool(n)

				np, ok := nps[npName]
				if !ok {
					names = append(names, npName)
					np = &nodepool{
						Name: npName,
						Type: instanceType(n),
					}
					nps[npName] = np
				}
				np.Nodes += 1
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)

			fmt.Fprintln(w, "NAME\tNODES\tTYPE")

			sort.Strings(names)
			for _, n := range names {
				np := nps[n]
				fmt.Fprintf(w, "%s\t%5d\t%s\n", np.Name, np.Nodes, np.Type)
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

			klient := ctx.Value("klient").(kubernetes.Interface)

			res, err := klient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			var ns []corev1.Node

			for _, n := range res.Items {
				if np := findNodepool(n); np == args[0] {
					ns = append(ns, n)
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)

			fmt.Fprintln(w, "NODE\tSTATUS")

			sort.Slice(ns, func(i, j int) bool { return ns[i].Name < ns[j].Name })
			for _, n := range ns {
				fmt.Fprintf(w, "%s\t%v\n", n.Name, nodeCondition(n))
			}

			return w.Flush()
		},
	}

	return cmd
}

func nodeCondition(n corev1.Node) string {
	var s strings.Builder

	for _, c := range n.Status.Conditions {
		if c.Status == corev1.ConditionTrue {
			if s.Len() > 0 {
				s.WriteRune(',')
			}
			s.WriteString(string(c.Type))
		}
	}

	return s.String()
}
