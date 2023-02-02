## EMQX Operator introduction

EMQX Enterprise is a cloud-native MQTT broker. EMQX Kubernetes Operator is a new way to create and manage cloud-native EMQX Enterprise instances based on Kubernetes architectures. It simplifies the process and required knowledge of the deployment and management.

**NOTE:** Only one EMQX Operator can be deployed in each Kubernetes cluster.

The corresponding relationship between EMQX Operator and EMQX version is as follows:

|      EMQX Version      |     EMQX Operator Version                            |     APIVersion    | 
|:----------------------:|:----------------------------------------------------:|:-----------------:| 
| 4.3.x <= EMQX < 4.4    | 1.2.0（stop maintenance）                             |  v1beta2          |
| 4.3.x <= EMQX < 4.4    | 1.2.1，1.2.2，1.2.3 （recommend）                      |  v1beta3          |
| 4.4.6 <= EMQX < 4.4.8  | 1.2.5                                                 |  v1beta3          | 
| 4.4.8 <= EMQX < 4.4.14 | 1.2.6，1.2.7，1.2.8，2.0.0，2.0.1，2.0.2 （recommend）  |  v1beta3          |
| 4.4.14 <= EMQX         | 2.1.0                                                 |  v1beta4          |
| 5.0.6 <= EMQX < 5.0.8  | 2.0.0，2.0.1 （recommend）                             |  v2alpha1         |
| 5.0.8 <= EMQX < 5.0.14 | 2.0.2                                                 |  v2alpha1         |
| 5.0.14 <= EMQX         | 2.1.0                                                 |  v2alpha1         |

## EMQX operator architecture
![](./introduction/assets/architecture.png)