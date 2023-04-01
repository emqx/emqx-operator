## Overview

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/), including EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of the EMQX cluster.

The EMQX Operator includes, but is not limited to, the following features:

* **Kubernetes Custom Resource**: Deploy and manage EMQX Cluster with pre-defined custom resources.

* **Simplified Deployment Configuration**: Configure the fundamentals of EMQX Cluster, including persistence, configuration, license, etc., in a Kubernetes-native way.

<img src="./introduction/assets/architecture.png" style="zoom:20%;" />

## EMQX and EMQX Operator compatibility

|      EMQX Version      |     EMQX Operator Version                            |     APIVersion    |
|:----------------------:|:----------------------------------------------------:|:-----------------:|
| 4.3.x <= EMQX < 4.4    | 1.2.1，1.2.2，1.2.3                                   |  v1beta3          |
| 4.4.6 <= EMQX < 4.4.8  | 1.2.5                                                 | v1beta3          |
| 4.4.8 <= EMQX < 4.4.14 | 1.2.6，1.2.7，1.2.8，2.0.0，2.0.1，2.0.2, 2.0.3        |  v1beta3          |
| 4.4.14 <= EMQX         | 2.1.0                                                 |  v1beta4          |
| 5.0.6 <= EMQX < 5.0.8  | 2.0.0，2.0.1, 2.0.3 .                                 |  v2alpha1         |
| 5.0.8 <= EMQX < 5.0.14 | 2.0.2                                                 |  v2alpha1         |
| 5.0.14 <= EMQX         | 2.1.0                                                 |  v2alpha1         |

## How to selector Kubernetes version

The EMQX Operator requires a Kubernetes cluster of version `>=1.24`.

+ Kubernetes version >= 1.24

  All functions of the EMQX and EMQX Operator can be used

+ Kubernetes version >= 1.21 && < 1.24

  The EMQX Operator can be used, but the [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) feature is not supported, it means in the `LoadBalancer` type of Service, the EMQX cluster can only use one protocol, such as TCP or UDP, but not both, so some features of EMQX will not be available.

+ Kubernetes version >= 1.20 && < 1.21

  The EMQX Operator can be used, but if using the `NodePort` type of Service, the user must manually assign the `.spec.ports[].nodePort`, otherwise every update to the Service will result in a change to the NodePort, For more details please refer to [Kubernetes changelog](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.20.md#bug-or-regression-4)

+ Kubernetes version >= 1.19 && < 1.20

  The EMQX Operator can be used, but we do not recommend using it, because the Kubernetes version is too old, we did not conduct a full test.

+ Kubernetes version < 1.19

  The EMQX Operator cannot be used, because the `discovery.k8s.io/v1` APIVersion is not supported.

