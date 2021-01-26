kim - The Kubernetes Image Manager
==================================

In ur kubernetes buildin' ur imagez.

***STATUS: EXPERIMENT - Let us know what you think***

This project is a continuation of the experiment started with `k3c`, however, unlike the original aim/design for `k3c`,
it ***IS NOT*** meant to be a replacement or re-build of the [containerd](https://containerd.io)/CRI.

`kim` is a Kubernetes-aware CLI that will install a small builder backend consisting of a [BuildKit](https://github.com/moby/buildkit)
daemon bound to the Kubelet's underlying containerd socket (for building images) along with a small server-side agent
that the CLI leverages for image management (think push, pull, etc) rather than talking to the backing containerd/CRI
directly. `kim` enables building images locally, natively on your [`k3s`](https://k3s.io) cluster.

## A familiar UX

There really is nothing better than the classic Docker UX of `build/push/pull/tag`.
This tool copies the same UX as classic Docker (think Docker v1.12). The intention
is to follow the same style but not be a 100% drop in replacement.  Behaviour and
arguments have been changed to better match the behavior of the Kubernetes ecosystem.

## A single binary

`kim`, similar to `k3s` and old school docker, is packaged as a single binary, because nothing is easier for
distribution than a static binary.

## Built on Kubernetes Tech (and others)

Fundamentally `kim` is a built on the [Container Runtime Interface (CRI)](https://github.com/kubernetes/cri-api),
[containerd](https://github.com/containerd/containerd), and [buildkit](https://github.com/moby/buildkit).

## Architecture

`kim` enables building `k3s`-local images by installing a DaemonSet Pod that runs both `buildkitd` and `kim agent`
and exposing the gRPC endpoints for these active agents in your cluster via a Service. Once installed, the `kim` CLI
can inspect your installation and communicate with the backend daemons for image building and manipulation with merely
the KUBECONFIG that was available when invoking `kim install`. When building `kim` will talk directly to the `buildkit`
service but all other interactions with the underlying containerd/CRI are mediated by the `kim agent` (primarily
because the `containerd` "smart client" code assumes a certain level of co-locality with the `containerd` installation).

## Building

```bash
# more to come on this front but builds are currently a very manual affair
# git clone --branch=trunk https://github.com/rancher/kim.git ~/Projects/rancher/kim
# cd ~/Projects/rancher/kim
go generate # only necessary when modifying the gRPC protobuf IDL, see Dockerfile for pre-reqs
make ORG=<your-dockerhub-org> build publish
```

## Running

Have a working `k3s` installation with a working `$HOME/.kube/config` or `$KUBECONFIG`, then:

```bash
# Installation on a single-node cluster
./bin/kim install --agent-image=docker.io/${ORG}/kim
```

```bash
# Installation on a multi-node cluster, targeting a Node named "my-builder-node"
./bin/kim install --agent-image=docker.io/${ORG}/kim --selector k3s.io/hostname=my-builder-node

```

`kim` currently works against a single builder Node so you must specify a narrow selector when
installing on multi-node clusters. Upon successful installation this node will acquire the "builder" role.

Build images like you would with the Docker CLI:

```
$ ./bin/kim --help
Kubernetes Image Manager -- in ur kubernetes buildin ur imagez

Usage:
  kim [OPTIONS] COMMAND
  kim [command]

Examples:
  kim build --tag your/image:tag .

Available Commands:
  build       Build an image
  help        Help about any command
  images      List images
  info        Display builder information
  install     Install builder component(s)
  pull        Pull an image
  push        Push an image
  rmi         Remove an image
  tag         Tag an image
  uninstall   Uninstall builder component(s)

Flags:
  -x, --context string      kubeconfig context for authentication
      --debug               
      --debug-level int     
  -h, --help                help for kim
  -k, --kubeconfig string   kubeconfig for authentication
  -n, --namespace string    namespace (default "kim")
  -v, --version             version for kim

Use "kim [command] --help" for more information about a command.
```

## Roadmap

- Automated builds for clients on MacOS (amd64/arm64), Windows (amd64), and Linux client/server (amd64/arm64/arm).

# License

Copyright (c) 2020-2021 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

