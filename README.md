# kubectl issues

`kubectl issues` is a kubectl plugin to find issues with your Kubernetes objects.

## Install

### Releases

Check the [releases](https://github.com/ricoberger/kubectl-issues/releases) page for the full list of pre-built binaries.

1. Download the release for your os/arch
2. Unzip to archive to get the `kubectl-issues` binary
3. Add the `kubectl-issues` binary to your `PATH`

### Source

```sh
go install github.com/ricoberger/kubectl-issues@latest
```

## Usage

```sh
kubectl issues
```

## Development

To build and run the binary the following commands can be used:

```sh
go build -o ./bin/kubectl-issues .
./bin/kubectl-issues
```

To publish a new version, a new tag must be created and pushed:

```sh
make release-patch
make release-minor
make release-major
```

## Acknowledgments

The plugin is inspired by the [`kubectl janitor`](https://github.com/dastergon/kubectl-janitor) plugin.
