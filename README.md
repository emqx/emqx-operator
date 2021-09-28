# emqx-operator

A Kubernetes Operator for EMQ X Broker

**Project status: *alpha*** Not all planned features are completed. The API, sepc, status and other user facing objects may chage, but in a backward compatible way.

Note: Project was previously known as emqx/emqx-operator.

## Overview

The EMQ X Operator provieds [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQ X](https://www.emqx.io/). The purpose of this project is to simplify and automate the configuration of EMQ X Broker for EMQ X cluster.

The EMQ X Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Use Kubernetes custom resource to deploy and manage EMQ X Broker.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQ X like persistence, conf, license from a native Kubernetes resource.

For an introduction to the EMQ X Operator, see the [getting started](docs/user-guides/getting-started.md) guide.

## Prerequisites

The EMQ X Operator requires a Kubernetes cluster of version `>=1.20.0`.If you are just starting out with the Emqx Operator, it is highly recommended to use the `version:v1.20.0`.

## CustomResourceDefinitions

A core feature of the EMQ X Operator is to monitor the Kubernetes API server for changes to specific objects and ensuer that the current Emqx deployments match thes objects.
The Operator acts on the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/):

* **`Emqx`**, which defines a desired EMQ X Broker Cluster deployment.

The EMQ X Operator automatically detects changes in the Kubernetes API server to any of the above objects, and ensures that matching deployments and configurations are kept in sync.

## Quickstart

For more information on quickstart, see the [user guide](docs/user-guides/getting-started-cn.md)

## Development

### Prerequisites

- golang environment
- docker (used for creating container images, etc.)
- Kubernetes cluster
  
## Contributing
Many files (api, config, controller, hack,...) in this repository are auto-generatoed. 
Before proposing a pull request:

1. Commit your changes.
2. Run `make` and `make manifests`
3. Commit the generated changes.

## Troubleshooting
Check the [troubleshooting documentation](docs/troubleshooting.md) for common issues and frequently asked questions (FAQ).