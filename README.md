# emqx-operator

A Kubernetes Operator for EMQ X Broker and EMQ X Enterprise

## Overview

The EMQ X Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQ X](https://www.emqx.io/) includes EMQ X Broker and EMQ X Enterprise. The purpose of this project is to simplify and automate the configuration of EMQ X cluster.

The EMQ X Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Deploy and manage EMQ X Cluster with pre-defined custom resources.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQ X Cluster, including persistence, configuration, license and etc, in a Kubernetes-native way.

For an introduction to the EMQ X Operator, see the [getting started](docs/user-guides/getting-started.md) guide.

## Prerequisites

The EMQ X Operator requires a Kubernetes cluster of version `>=1.20.0`.If you are just starting out with the EmqxBroker Operator, it is highly recommended to use the `version:v1.20.0`.

## CustomResourceDefinitions

A core feature of the EMQ X Operator is to monitor the Kubernetes API server for changes to specific objects and ensure that the running EMQ X deployments match these objects.
The Operator acts on the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/).

The example of EMQ X Broker see the [emqx.yaml](config/samples/emqx/emqx.yaml)

The EMQ X Operator automatically detects changes on any of the above custom resource objects, and ensures that running deployments are kept in sync with the changes.

## Quickstart

For more information on quickstart, see the [user guide](docs/user-guides/getting-started.md)

## Development

### Prerequisites

- golang environment
- docker (used for creating container images, etc.)
- Kubernetes cluster
  
## Contributing
Many files (api, config, controller, hack,...) in this repository are auto-generated. 
Before proposing a pull request:

1. Commit your changes.
2. Run `make` and `make manifests`
3. Commit the generated changes.

## Troubleshooting
Check the [troubleshooting documentation](docs/troubleshooting.md) for common issues and frequently asked questions (FAQ).