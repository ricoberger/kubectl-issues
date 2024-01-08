# kubectl issues

`kubectl issues` is a kubectl plugin to find issues with your Kubernetes objects. The plugin is inspired by the [`kubectl janitor`](https://github.com/dastergon/kubectl-janitor) plugin.

## Install

### Releases

Check the [releases](https://github.com/ricoberger/kubectl-issues/releases) page for the full list of pre-built binaries.

1. Download the release for your os/arch
2. Unzip to archive to get the `kubectl-issues` binary
3. Add the `kubectl-issues` binary to your `PATH`

### Source

```sh
go install github.com/ricoberger/kubectl-issues/cmd@latest
```

## Usage

```sh
kubectl issues
```
