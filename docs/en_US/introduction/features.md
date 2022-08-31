# Main features

- Quickly deploy EMQX Enterprise cluster without complicated configuration
- Rolling update of EMQX Enterprise cluster based on changes in definition
- Automatic scaling of EMQX Enterprise
- Auto discover EMQX Listeners, and bind listeners port to kubernetes service
- When updating EMQX Plugin Custom Resources, like change listener port, or update plugin configure, it will not restart Pods
- Upgrade the EMQX Enterprise version without interrupting the service
- Monitoring of clusters and nodes can be integrated with Prometheus
- Official EMQX Enterprise container image is integrated
- New stateless node: EMQX Replicant, use Deployment((EMQX Operator >= 2.0))