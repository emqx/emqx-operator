# Access EMQX Cluster by LoadBalancer

## Task Target

How to access EMQX cluster by LoadBalancer. <!--I do not quite understand what is page is about-->

## Configure EMQX Cluster

Here are the relevant configurations for EMQX Custom Resource. You can choose the corresponding APIVersion based on the version of EMQX you wish to deploy. For specific compatibility relationships, please refer to [EMQX Operator Compatibility](../README.md):

:::: tabs type:card
::: tab v2alpha1

EMQX CRD supports using `.spec.dashboardServiceTemplate` to configure EMQX cluster Dashboard Service, using `.spec.listenersServiceTemplate` to configure EMQX cluster listener Service, its documentation can refer to [Service](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v2alpha1-reference.md).

```yaml
apiVersion: apps.emqx.io/v2alpha1
kind: EMQX
metadata:
  name: emqx
spec:
  image: emqx:5.0
  listenersServiceTemplate:
    spec:
      type: LoadBalancer
```

> By default, EMQX will open an MQTT TCP listener `tcp-default` corresponding to port 1883 and Dashboard listener `dashboard-listeners-http-bind` corresponding to port 18083. Users can add new listeners through `.spec.bootstrapConfig` field or EMQX Dashboard. EMQX Operator will automatically inject the default listener information into the Service when creating the Service, but when there is a conflict between the Service configured by the user and the listener configured by EMQX (name or port fields are repeated), EMQX Operator will use the user's configuration prevail.

Save the above content as `emqx.yaml` and execute the following command to deploy the EMQX cluster:

```bash
$ kubectl apply -f emqx.yaml

emqx.apps.emqx.io/emqx created
```

Check the status of the EMQX cluster and make sure that `STATUS` is `Running`, which may take some time to wait for the EMQX cluster to be ready.

```bash
$ kubectl get emqx emqx

NAME   IMAGE      STATUS    AGE
emqx   emqx:5.0   Running   10m
```

:::
::: tab v1beta4

The corresponding CRD of EMQX Enterprise in EMQX Operator is EmqxEnterprise, and EmqxEnterprise supports configuring EMQX cluster Service through `.spec.serviceTemplate` field. For the specific description of the serviceTemplate field, please refer to [serviceTemplate](https://github.com/emqx/emqx-operator/blob/main-2.1/docs/en_US/reference/v1beta4-reference.md#servicetemplate).

```yaml
apiVersion: apps.emqx.io/v1beta4
kind: EmqxEnterprise
metadata:
  name: emqx-ee
spec:
  template:
    spec:
      emqxContainer:
        image:
          repository: emqx/emqx-ee
          version: 4.4.14
  serviceTemplate:
    spec:
      type: LoadBalancer
```

> EMQX will open 6 listeners by default, namely: `mqtt-ssl-8883` corresponds to port 8883, `mqtt-tcp-1883` corresponds to port 1883, `http-dashboard-18083` corresponds to port 18083, `http-management-8081` corresponds to port 8081,`mqtt-ws-8083` corresponds to port 8083 and `mqtt-wss-8084` corresponds to port 8084. EMQX Operator will automatically inject the default listener information into the Service when creating the Service, but when there is a conflict between the Service configured by the user and the listener configured by EMQX (the name or port field is repeated), EMQX Operator will use the user's configuration prevail.

Save the above content as `emqx.yaml` and execute the following command to deploy the EMQX cluster:

```bash
$ kubectl apply -f emqx.yaml

emqxenterprise.apps.emqx.io/emqx-ee created
```

Check the status of the EMQX cluster and make sure that `STATUS` is `Running`, which may take some time to wait for the EMQX cluster to be ready.

```bash
$ kubectl get emqxenterprises

NAME      STATUS   AGE
emqx-ee   Running  8m33s
```

:::
::::

## Use MQTT X Client to Connect EMQX Cluster

Check EMQX service

```bash
$ kubectl get svc -l apps.emqx.io/instance=emqx

NAME             TYPE       CLUSTER-IP       EXTERNAL-IP            PORT(S)                          AGE
emqx-dashboard   NodePort   10.101.225.238   183.134.197.178        18083:32012/TCP                  32s
emqx-listeners   NodePort   10.97.59.150     183.134.197.178        1883:32010/TCP                   10s
```

use MQTT X Cli to connect EMQX

```
$ mqttx conn -h 183.134.197.178
[11:16:40] › …  Connecting...
[11:16:41] › ✔  Connected
```

## Add New listeners via EMQX Dashboard

Open the browser, enter the host `IP` and port `32012` where the EMQX Pod is located, log in to the EMQX cluster Dashboard (Dashboard default user name: admin, default password: public), enter the Dashboard and click Configuration → Listeners to enter the listener page, We first click the Add Listener button to add a listener named to test and port 1884, as shown in the figure below:

<img src="./assets/configure-service/emqx-add-listener.png" style="zoom: 33%;" />

Then click the Add button to create the listener, as shown in the following figure:

<img src="./assets/configure-service/emqx-listeners.png" style="zoom:50%;" />

As can be seen from the figure, the test listener we created has taken effect.

- Check whether the newly added listener is injected into the Service

```bash
kubectl get svc -l apps.emqx.io/instance=emqx
```

The output is similar to:

```bash
NAME             TYPE       CLUSTER-IP       EXTERNAL-IP   PORT(S)                                         AGE
emqx-dashboard   NodePort   10.105.110.235   <none>        18083:32012/TCP                                 13m
emqx-listeners   NodePort   10.106.1.58      <none>        1883:32010/TCP,14567:32011/UDP,1884:30763/TCP   12m
```

From the output results, we can see that the newly added listener 1884 has been injected into the `emqx-listeners` Service.
