# kubectl-nodepools
A `kubectl` plugin for listing node pools/groups.

## Installation
Make sure your Go bin directory is in your `PATH`:

```shell
export PATH="$(go env GOPATH)/bin:${PATH}"
```

You can install `kubectl-nodepools` using [`krew`](https://krew.sigs.k8s.io/) with the following command:

```shell
kubectl krew install nodepools
```

You can also install it using the following command:

```
go install github.com/grafana/kubectl-nodepools@latest
```

## Usage
`kubectl-nodepools` accepts the default flags from `kubectl`, like `--context`.
Pass `--help` for additional help.

### Listing node pools/groups
```shell
kubectl nodepools list
```

### Listing nodes in a given node pool/group
```shell
kubectl nodepools nodes $nodepool
```

Where `$nodepool` should be the name of an existing node pool.

### Using a custom node pool label
If your cluster uses a different label than the ones supported in code, you can pass a custom label using the `--label/-l` flag or by setting the `KUBE_NODEPOOLS_LABEL` environment variable:

```shell
# list nodepools using a custom label
kubectl nodepools list --label 'custom.domain.io/fancy-node-label'

# list nodes with a nodepool using a custom label
kubectl nodepools nodes -l 'custom.domain.io/fancy-node-label' $nodepool

# using environment variable
export KUBE_NODEPOOLS_LABEL="custom.domain.io/fancy-node-label"
kubectl nodepools list
kubectl nodepools nodes $nodepool
```

### Working with Karpenter
Because [Karpenter](https://karpenter.sh/) does not provision node groups, it must be handled separately.
By default, Karpenter nodes are listed in the format "(Karpenter) {Provisioner}".
The [Provisioner](https://karpenter.sh/v0.27.1/concepts/provisioners/) refers to the provisioner that Karpenter used to create this particular node.
In order to search for nodes from particular provisioners, use the `--label` flag.

```shell
# list nodes with karpenter included
kubectl nodepools list

# list nodes for a particular karpenter provisioner
kubectl nodepools nodes --label 'karpenter.sh/provisioner-name' provisionerA
```
