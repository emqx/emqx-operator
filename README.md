# emqx-operator

A Kubernetes Operator for EMQX Broker and EMQX Enterprise

## Overview

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/) includes EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of EMQX cluster.

The EMQX Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Deploy and manage EMQX Cluster with pre-defined custom resources.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQX Cluster, including persistence, configuration, license and etc, in a Kubernetes-native way.

For an introduction to the EMQX Operator, see the [introduction](docs/en_US/README.md).

## Prerequisites

The EMQX Operator requires a Kubernetes cluster of version `>=1.20.0`.

## CustomResourceDefinitions

A core feature of the EMQX Operator is to monitor the Kubernetes API server for changes to specific objects and ensure that the running EMQX deployments match these objects.
The Operator acts on the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/).

The example of EMQX Broker see the [emqx.yaml](config/samples/emqx/v1beta2/emqx.yaml) and the example of EMQX Enterprise see the [emqx-ee.yaml](config/samples/emqx/v1beta2/emqx-ee.yaml).

The EMQX Operator automatically detects changes on any of the above custom resource objects, and ensures that running deployments are kept in sync with the changes.

## Getting Start

For more information on get started, see the [getting started](docs/en_US/getting-started/getting-started.md)

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
Check the [troubleshooting documentation](docs/en_US/faq/faq.md) for common issues and frequently asked questions (FAQ).
