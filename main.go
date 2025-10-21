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

const (
	karpenterNodePrefix string = "(Karpenter) "
	customLabelEnvVar   string = "KUBE_NODEPOOLS_LABEL"
)

var karpenterLabels = []string{"karpenter.sh/provisioner-name", "karpenter.sh/nodepool"}

var (
	noHeaders bool
	onlyName  bool
	output    string
	label     string
)

func main() {
	cmd := rootCmd()
	cmd.AddCommand(listCmd())
	cmd.AddCommand(nodesCmd())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	err := cmd.ExecuteContext(ctx)
	cancel()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type ctxKey string

var kubeClientKey = ctxKey("klient")

func rootCmd() *cobra.Command {
	kflags := genericclioptions.NewConfigFlags(true)

	cmd := &cobra.Command{
		Use:   "nodepools",
		Short: "Read-only interaction with nodepools",
		Long: `Read-only interaction with nodepools.

List node pools/groups in the current cluster, alongside a count of
how many nodes there are in each pool/group and their type.

You can also list nodes for a given node pool/group by name.`,
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

			ctx = context.WithValue(ctx, kubeClientKey, klient)

			cmd.SetContext(ctx)

			if output != "" && output != "name" {
				return fmt.Errorf("unrecognized --output type %s, only name is valid", output)
			}

			onlyName = output == "name"

			return nil
		},
		SilenceErrors: true,
	}

	flags := cmd.PersistentFlags()
	flags.BoolVar(&noHeaders, "no-headers", false, "Don't print headers (default print headers)")
	flags.StringVarP(&output, "output", "o", "", "Output format. Only name.")
	labelHelp := fmt.Sprintf("Label to group nodes into pools with; can be set via %s environment variable", customLabelEnvVar)
	flags.StringVarP(&label, "label", "l", os.Getenv(customLabelEnvVar), labelHelp)
	kflags.AddFlags(flags)

	return cmd
}

var providerNodepoolLabels = map[string]string{
	"AWS":  "eks.amazonaws.com/nodegroup",
	"GCP":  "cloud.google.com/gke-nodepool",
	"AKS":  "kubernetes.azure.com/agentpool",
	"DOKS": "doks.digitalocean.com/node-pool-id",
}

func findNodepool(node corev1.Node, label string) string {
	// Check the custom label first
	if np, ok := node.Labels[label]; ok {
		return np
	}

	// check for karpenter nodes
	for _, label := range karpenterLabels {
		if np, ok := node.Labels[label]; ok {
			return karpenterNodePrefix + np
		}
	}

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
	Types map[string]int
	Nodes uint
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List node pools/groups in current cluster",
		Long:  `List node pools/groups in the current cluster, alongside a count of nodes and their type.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			warnEnvLabelUsage(cmd)

			ctx := cmd.Context()

			klient := ctx.Value(kubeClientKey).(kubernetes.Interface)

			res, err := klient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			nps := make(map[string]*nodepool)
			names := make([]string, 0, len(nps))

			for _, n := range res.Items {
				npName := findNodepool(n, label)

				np, ok := nps[npName]
				if !ok {
					names = append(names, npName)
					np = &nodepool{
						Name: npName,
						Types: map[string]int{
							instanceType(n): 1,
						},
					}
					nps[npName] = np
				} else {
					np.Types[instanceType(n)]++
				}
				np.Nodes += 1
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

			if !noHeaders {
				if onlyName {
					fmt.Fprintln(w, "NAME") // nolint:errcheck
				} else {
					fmt.Fprintln(w, "NAME\tNODES\tTYPE") // nolint:errcheck
				}
			}

			sort.Strings(names)
			for _, n := range names {
				np := nps[n]
				if onlyName {
					fmt.Fprintln(w, np.Name) // nolint:errcheck
				} else {
					typeList := make([]string, 0, len(np.Types))
					for k, v := range np.Types {
						typeWithCount := fmt.Sprintf("%s (%d)", k, v)
						typeList = append(typeList, typeWithCount)
					}
					sort.Strings(typeList)
					fmt.Fprintf(w, "%s\t%5d\t%s\n", np.Name, np.Nodes, strings.Join(typeList, ", ")) // nolint:errcheck
				}
			}

			return w.Flush()
		},
		Aliases: []string{"ls"},
	}

	return cmd
}

func nodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes <name>",
		Short: "List nodes in node pool/group",
		Long:  `List nodes in the given node pool/group, alongside their status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("need to pass a single nodepool name")
			}

			warnEnvLabelUsage(cmd)

			ctx := cmd.Context()

			klient := ctx.Value(kubeClientKey).(kubernetes.Interface)

			res, err := klient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			var (
				ns   []corev1.Node
				name = strings.TrimPrefix(args[0], karpenterNodePrefix)
			)

			for _, n := range res.Items {
				if np := findNodepool(n, label); strings.TrimPrefix(np, karpenterNodePrefix) == name {
					ns = append(ns, n)
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

			if !noHeaders {
				if onlyName {
					fmt.Fprintln(w, "NODE") // nolint:errcheck
				} else {
					fmt.Fprintln(w, "NODE\tSTATUS") // nolint:errcheck
				}
			}

			sort.Slice(ns, func(i, j int) bool { return ns[i].Name < ns[j].Name })
			for _, n := range ns {
				if onlyName {
					fmt.Fprintln(w, n.Name) // nolint:errcheck
				} else {
					fmt.Fprintf(w, "%s\t%v\n", n.Name, nodeCondition(n)) // nolint:errcheck
				}
			}

			return w.Flush()
		},
		Aliases: []string{"ns"},
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

func warnEnvLabelUsage(cmd *cobra.Command) {
	if label != "" && !cmd.Parent().PersistentFlags().Changed("label") {
		fmt.Fprintf(os.Stderr, "Using custom label %q set by environment variable %s\n", label, customLabelEnvVar)
	}
}
