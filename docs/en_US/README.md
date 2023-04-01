## Overview

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/), including EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of the EMQX cluster.

The EMQX Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Deploy and manage EMQX Cluster with pre-defined custom resources.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQX Cluster, including persistence, configuration, license, etc., in a Kubernetes-native way.

<img src="./introduction/assets/architecture.png" style="zoom:20%;" />

## EMQX and EMQX Operator compatibility

|      EMQX Version      |     EMQX Operator Version                            |     APIVersion    |
|:----------------------:|:----------------------------------------------------:|:-----------------:|
| 4.3.x (included) ～ 4.4 | 1.2.1, 1.2.2, 1.2.3                                 |  v1beta3          |
| 4.4.6 (included) ～ 4.4.8 | 1.2.5                                                 | v1beta3          |
| 4.4.8 (included) ～ 4.4.14 | 1.2.6, 1.2.7, 1.2.8, 2.0.0, 2.0.1, 2.0.2, 2.0.3   |  v1beta3          |
| 4.4.14 or higher 4.4.x | 2.1.0                                                 |  v1beta4          |
| 5.0.6 (included) ～ 5.0.8 | 2.0.0, 2.0.1, 2.0.3 .                                |  v2alpha1         |
| 5.0.8 (included) ～  5.0.14 | 2.0.2                                                 |  v2alpha1         |
| 5.0.14 or higher | 2.1.0                                                 |  v2alpha1         |

## How to selector Kubernetes version

The EMQX Operator requires a Kubernetes cluster of version `>=1.24`.



| Kubernetes Versions     | EMQX Operator Compatibility                                  | Notes                                                        |
| ----------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| 1.24 or higher          | All functions supported                                      |                                                              |
| 1.21 (included) ～ 1.23 | Supported, except [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) | EMQX cluster can only use one protocol in `LoadBalancer` type of Service, for example TCP or UDP. |
| 1.20 (included) ～ 1.21 | Supported, manual `.spec.ports[].nodePort` assignment required if using `NodePort` type of Service | For more details, please refer to [Kubernetes changelog](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.20.md#bug-or-regression-4). |
| 1.19 (included) ～ 1.20 | Supported, not recommended due to lack of testing            |                                                              |
| Lower than 1.19         | Not supported                                                | `discovery.k8s.io/v1` APIVersion not supported               |

