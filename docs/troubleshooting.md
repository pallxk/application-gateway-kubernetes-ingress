# Troubleshooting [WIP]

* [How to view Ingress Controller logs](#how-to-view-ingress-controller-logs)
* [How to view Ingress Controller events](#how-to-view-ingress-Controller-events)
* [How do view current Application Gateway configuration](#how-do-view-current-Application-Gateway-configuration)
* [How do enable verbose logging](#how-do-enable-verbose-logging)

## How to view Ingress Controller logs

1) List the installed helm packages using `helm list`. Find the AGIC chart. We will use the chart name and namespace to the look up the AGIC controller pod.

```bash
$> helm list
NAME            REVISION        UPDATED                         STATUS          CHART                   APP VERSION     NAMESPACE
odd-billygoat   1               xx        DEPLOYED        ingress-azure-0.6.0     0.6.0           default
```

2) Fing `Pod` matching with helm chart name in the namespace. Here, We look for `odd-billygoat` in `default` namespace

```bash
$> kubectl get pods --namespace=default | grep "odd-billygoat"
odd-billygoat-ingress-azure-584597fbf-c8trb   1/1       Running   0          32m
```

3) Use kubectl logs <controller-pod-name> to view the logs.

```bash
$> kubectl logs odd-billygoat-ingress-azure-584597fbf-c8trb
I0621 21:16:07.253653       1 main.go:113] Creating authorizer from Azure Managed Service Identity
I0621 21:16:09.907672       1 context.go:293] k8s context run started
...
```

If you have a single instance of the ingress controller running, then you can also leverage the following command.

```bash
$> kubectl get pods --namespace=default --selector="app=ingress-azure" --output="NAME" | xargs kubectl logs
I0621 21:16:07.253653       1 main.go:113] Creating authorizer from Azure Managed Service Identity
I0621 21:16:09.907672       1 context.go:293] k8s context run started
...
```

## How to view Ingress Controller events

Ingress controller publishes events to the Ingress resources when it finds issues with it's configuration.

Following is an example where `ingress/sample-app` is referencing a `service/sample-app` that doesn't exists.

```bash
$> kubectl describe ingress/sample-app
Name:             sample-app
Namespace:        default
Address:          
Default backend:  default-http-backend:80 (<none>)
Rules:
Host  Path  Backends
----  ----  --------
*     
        sample-app:80 (<none>)
Annotations:
kubectl.kubernetes.io/last-applied-configuration:  {"apiVersion":"extensions/v1beta1","kind":"Ingress","metadata":{"annotations":{"kubernetes.io/ingress.class":"azure/application-gateway"},"name":"sample-app","namespace":"default"},"spec":{"rules":[{"http":{"paths":[{"backend":{"serviceName":"sample-app","servicePort":80}}]}}]}}

kubernetes.io/ingress.class:  azure/application-gateway
Events:
Type     Reason           Age                 From                                                                    Message
----     ------           ----                ----                                                                    -------
Warning  ServiceNotFound  19m                 azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-q2lhv  Unable to get the service [default/sample-app]
Warning  EndpointsEmpty   19m                 azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-q2lhv  Unable to get endpoints for service key [default/sample-app]
Warning  ServiceNotFound  16m                 azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-tdsqp  Unable to get the service [default/sample-app]
Warning  EndpointsEmpty   16m                 azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-tdsqp  Unable to get endpoints for service key [default/sample-app]
Warning  ServiceNotFound  15m                 azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-j5pd8  Unable to get the service [default/sample-app]
Warning  EndpointsEmpty   15m                 azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-j5pd8  Unable to get endpoints for service key [default/sample-app]
Warning  ServiceNotFound  14s (x13 over 17s)  azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-c8trb  Unable to get the service [default/sample-app]
Warning  EndpointsEmpty   14s (x12 over 17s)  azure/application-gateway, odd-billygoat-ingress-azure-584597fbf-c8trb  Unable to get endpoints for service key [default/sample-app]
```

## How do view current Application Gateway configuration

Install `az cli`.

```bash
az network application-gateway show  -n <applicationGatewayName> -g <resourceGroup>
```

Another way is to view Application gateway configuration is to set [ingress controller log verbosity](#how-do-enable-verbose-logging) to 5 by modifying the helm config.

## How do enable verbose logging

To increase the log verbosity, modify the helm config by adding:

```yaml
verbosityLevel: 5
```
