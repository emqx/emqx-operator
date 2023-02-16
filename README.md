# EMQX Operator

[![GitHub Release](https://img.shields.io/github/release/emqx/emqx-operator?color=brightgreen)](https://github.com/emqx/emqx-operator/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/emqx/emqx-operator-controller)](https://hub.docker.com/r/emqx/emqx-operator-controller)
[![codecov](https://codecov.io/gh/emqx/emqx-operator/branch/main/graph/badge.svg?token=RNMH7K52JZ)](https://codecov.io/gh/emqx/emqx-operator)

A Kubernetes Operator for [EMQX](https://www.emqx.io)

## Overview

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/), including EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of the EMQX cluster.

The EMQX Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Deploy and manage EMQX Cluster with pre-defined custom resources.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQX Cluster, including persistence, configuration, license, etc., in a Kubernetes-native way.

For an introduction to the EMQX Operator, see the [introduction](docs/en_US/README.md).

## Prerequisites

The EMQX Operator requires a Kubernetes cluster of version `>=1.24`.

### How to selector Kubernetes version

+ Kubernetes version >= 1.24

  All functions of the EMQX and EMQX Operator can be used

+ Kubernetes version >= 1.21 && < 1.24

  The EMQX Operator can be used, but the [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) feature is not supported, it means in `LoadBalancer` type of Service, the EMQX cluster can only use one protocol, such as TCP or UDP, but not both, so some features of EMQX will not be available.

+ Kubernetes version >= 1.20 && < 1.21

  The EMQX Operator can be used, but if use `NodePort` type of Service, user must manually assign the `.spec.ports[].nodePort`, otherwise every update to the Service will result in a change to the NodePort, more details please refer to [Kubernetes changelog](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.20.md#bug-or-regression-4)

+ Kubernetes version >= 1.16 && < 1.20

  The EMQX Operator can be used, but we do not recommend using it, because the Kubernetes version is too old, we did not conducted a full test.

+ Kubernetes version < 1.16

  The EMQX Operator cannot be used, because the `apiextensions/v1` APIVersion is not supported.

## CustomResourceDefinitions

A core feature of the EMQX Operator is to monitor the Kubernetes API server for changes to specific objects and ensure that the running EMQX deployments match these objects.
The Operator acts on the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/).

For the example of EMQX, see the [`emqx-full.yaml`](config/samples/emqx/v2alpha1/emqx-full.yaml).

The EMQX Operator automatically detects changes on any of the above custom resource objects and ensures that running deployments are kept in sync with the changes.

## EMQX Operator compatibility

|                   | EMQX 4.2 | EMQX 4.3 | EMQX 4.4 | EMQX 5.0 |
|-------------------|----------|----------|----------| ---------|
| EMQX Operator 1.1 | ✓        | ✓        |          |          |
| EMQX Operator 1.2 |          |          | ✓        |          |
| EMQX Operator 2.0 |          |          | ✓        | ✓        |
| EMQX Operator 2.1 |          |          | ✓        | ✓        |

## Getting Start

For more information on getting started, see the [getting started](docs/en_US/getting-started/getting-started.md).

## Public Cloud Platform Deployment Guide

|  Public Cloud Platform   | Deployment Guide                                         |
|--------------------------|----------------------------------------------------------|
|    AWS                   | [EKS](docs/en_US/deployment/aws-eks-deployment.md)       |
|    Azure                 | [AKS](docs/en_US/deployment/azure-deployment.md)       |
|    Google Cloud          | [GKE](docs/en_US/deployment/gcp-gke-deployment.md)       |
|    Alibaba Cloud         | [ACK](docs/zh_CN/deployment/aliyun-ack-deployment.md)    |
|    Huawei                | [CCE](docs/zh_CN/deployment/cce-deployment.md)           |
|    Tencent               | [TKE](docs/zh_CN/deployment/tencent-tke-deployment.md)   |


## Development

### Prerequisites

- Golang environment
- docker (used for creating container images, etc.)
- Kubernetes cluster

## Contributing
Many files (API, config, controller, hack,...) in this repository are auto-generated.
Before proposing a pull request:

1. Commit your changes.
2. `make` and `make manifests`
3. Commit the generated changes.

## Troubleshooting
Check the [troubleshooting documentation](docs/en_US/faq/faq.md) for common issues and frequently asked questions (FAQ).
