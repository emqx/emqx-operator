## Overview

The EMQX Operator provides [Kubernetes](https://kubernetes.io/) native deployment and management of [EMQX](https://www.emqx.io/), including EMQX Broker and EMQX Enterprise. The purpose of this project is to simplify and automate the configuration of the EMQX cluster.

The EMQX Operator includes, but is not limited to, the following features:

* **Simplified Deployment EMQX**: Declare EMQX clusters with EMQX custom resources and deploy them quickly. For more details, please check [Quick Start](./getting-started/getting-started.md).

* **Manage EMQX Cluster**: Automate operations and maintenance for EMQX, including cluster upgrades, runtime data persistence, updating Kubernetes resources based on the status of EMQX, etc. For more details, please check [Manage EMQX](./tasks/overview.md).

<img src="./introduction/assets/architecture.png" style="zoom:20%;" />

## EMQX and EMQX Operator compatibility

### EMQX Enterprise

|  EMQX Enterprise Version   |              EMQX Operator Version              |                          APIVersion                          |      Kind      |
| :------------------------: | :---------------------------------------------: | :----------------------------------------------------------: | :------------: |
|  4.3.x (included) ～ 4.4   |               1.2.1, 1.2.2, 1.2.3               | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md) | EmqxEnterprise |
| 4.4.6 (included) ～ 4.4.8  |                      1.2.5                      | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md) | EmqxEnterprise |
| 4.4.8 (included) ～ 4.4.14 | 1.2.6, 1.2.7, 1.2.8, 2.0.0, 2.0.1, 2.0.2, 2.0.3 | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md) | EmqxEnterprise |
|   4.4.14 or higher 4.4.x   |                  2.1.0, 2.1.1, 2.1.2                   | [apps.emqx.io/v1beta4](./reference/v1beta4-reference.md) | EmqxEnterprise |
|      5.0.0 or higher       |    2.0.0, 2.0.1, 2.0.2, 2.0.3, 2.1.0, 2.1.1, 2.1.2     | [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md) |      EMQX      |

### EMQX Open Source

|      EMQX Open Source Version      |     EMQX Operator Version                            |     APIVersion    |    Kind    |
|------------------------|-------------------|-------------------|-------------------|
| 4.3.x (included) ～ 4.4 | 1.2.1, 1.2.2, 1.2.3                                 |  [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md)          |  EmqxBroker  |
| 4.4.6 (included) ～ 4.4.8 | 1.2.5                                                 | [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md)          | EmqxBroker |
| 4.4.8 (included) ～ 4.4.14 | 1.2.6, 1.2.7, 1.2.8, 2.0.0, 2.0.1, 2.0.2, 2.0.3   |  [apps.emqx.io/v1beta3](./reference/v1beta3-reference.md)          | EmqxBroker |
| 4.4.14 or higher 4.4.x | 2.1.0, 2.1.1, 2.1.2                                                 |  [apps.emqx.io/v1beta4](./reference/v1beta4-reference.md)          | EmqxBroker |
| 5.0.6 (included) ～ 5.0.8 | 2.0.0, 2.0.1, 2.0.3                                |  [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md)         |  EMQX     |
| 5.0.8 (included) ～  5.0.14 | 2.0.2                                            |  [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md)         |  EMQX     |
| 5.0.14 or higher | 2.1.0, 2.1.1, 2.1.2                                                | [apps.emqx.io/v2alpha1](./reference/v2alpha1-reference.md)         | EMQX     |

## How to selector Kubernetes version

The EMQX Operator requires a Kubernetes cluster of version `>=1.24`.

| Kubernetes Versions     | EMQX Operator Compatibility                                  | Notes                                                        |
| ----------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| 1.24 or higher          | All functions supported                                      |                                                              |
| 1.21 (included) ～ 1.23 | Supported, except [MixedProtocolLBService](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) | EMQX cluster can only use one protocol in `LoadBalancer` type of Service, for example TCP or UDP. |
| 1.20 (included) ～ 1.21 | Supported, manual `.spec.ports[].nodePort` assignment required if using `NodePort` type of Service | For more details, please refer to [Kubernetes changelog](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-1.20.md#bug-or-regression-4). |
| 1.16 (included) ～ 1.20 | Supported, not recommended due to lack of testing            |                                                              |
| Lower than 1.16         | Not supported                                                | `apiextensions/v1` APIVersion is not supported.               |
