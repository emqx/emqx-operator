# EMQX Operator

[![GitHub Release](https://img.shields.io/github/release/emqx/emqx-operator?color=brightgreen)](https://github.com/emqx/emqx-operator/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/emqx/emqx-operator-controller)](https://hub.docker.com/r/emqx/emqx-operator-controller)
[![Coverage Status](https://coveralls.io/repos/github/emqx/emqx-operator/badge.svg?branch=main)](https://coveralls.io/github/emqx/emqx-operator?branch=main)

A Kubernetes Operator for [EMQX](https://www.emqx.io)

## Overview

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/), including EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of the EMQX cluster.

The EMQX Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Deploy and manage EMQX Cluster with pre-defined custom resources.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQX Cluster, including persistence, configuration, license, etc., in a Kubernetes-native way.

For an introduction to the EMQX Operator, see the [introduction](docs/en_US/README.md).

## Prerequisites

The EMQX Operator requires a Kubernetes cluster of version `>=1.24`.

> ### Why we need kubernetes 1.24:
>
> The `MixedProtocolLBService` feature is enabled by default in Kubernetes 1.24 and above. For its documentation, please refer to: [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/). The `MixedProtocolLBService` attribute allows different protocols to be used within the same Service instance of type `LoadBalancer`. Therefore, if the user deploys the EMQX cluster on Kubernetes and uses the `LoadBalancer` type of Service, there are both TCP and UDP protocols in the Service, please pay attention to upgrading the Kubernetes version to 1.24 or above, otherwise the Service creation will fail.
> 
> **If user doesn't need `MixedProtocolLBService` feature, the EMQX Operator requires a Kubernetes cluster of version `>=1.21`.**

## CustomResourceDefinitions

A core feature of the EMQX Operator is to monitor the Kubernetes API server for changes to specific objects and ensure that the running EMQX deployments match these objects.
The Operator acts on the following [custom resource definitions (CRDs)](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/).

For the example of EMQX, see the [`emqx-full.yaml`](config/samples/emqx/v2alpha1/emqx-full.yaml).

The EMQX Operator automatically detects changes on any of the above custom resource objects and ensures that running deployments are kept in sync with the changes.

## EMQX Operator compatibility

|                          | EMQX 4.2 latest | EMQX 4.3 latest | EMQX 4.4 latest | EMQX 5.0 latest |
|--------------------------|-----------------|-----------------|-----------------| ----------------|
| EMQX Operator 1.1 latest | ✓               | ✓               |                 |                 |
| EMQX Operator 1.2 latest |                 |                 | ✓               |                 |
| EMQX Operator 2.0 latest |                 |                 | ✓               | ✓               |

## Getting Start

For more information on getting started, see the [getting started](docs/en_US/getting-started/getting-started.md).

## Public Cloud Platform Deployment Guide

|  Public Cloud Platform   | Deployment Guide                                         |
|--------------------------|----------------------------------------------------------|
|    AWS                   | [EKS](docs/en_US/deployment/aws-eks-deployment.md)       |
|    Azure                 | [Azure](docs/en_US/deployment/azure-deployment.md)       |
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
