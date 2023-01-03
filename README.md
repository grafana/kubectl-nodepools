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
