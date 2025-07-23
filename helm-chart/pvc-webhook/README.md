# PVC Webhook Helm Chart

This Helm chart deploys the PVC Size Validation Webhook to a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- cert-manager installed in the cluster

## Installing cert-manager

If cert-manager is not already installed in your cluster:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
```

## Installation

1. **Install the chart with default values:**
   ```bash
   helm install pvc-webhook ./helm-chart/pvc-webhook
   ```

2. **Install with custom values:**
   ```bash
   helm install pvc-webhook ./helm-chart/pvc-webhook \
     --set config.maxPvcSizeBytes=10737418240 \
     --set replicaCount=2
   ```

3. **Install with a custom values file:**
   ```bash
   helm install pvc-webhook ./helm-chart/pvc-webhook -f my-values.yaml
   ```

## Configuration

The following table lists the configurable parameters and their default values:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of webhook replicas | `1` |
| `image.repository` | Container image repository | `sebiuo/pvc-webhook` |
| `image.tag` | Container image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `Always` |
| `config.maxPvcSizeBytes` | Maximum PVC size in bytes | `5368709120` (5 GiB) |
| `config.port` | Webhook server port | `9093` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `443` |
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `autoscaling.enabled` | Enable horizontal pod autoscaling | `false` |
| `autoscaling.minReplicas` | Minimum number of replicas | `1` |
| `autoscaling.maxReplicas` | Maximum number of replicas | `100` |
| `namespace` | Namespace for deployment | `cluster-tools` |
| `certificate.enabled` | Enable certificate management | `true` |
| `webhook.enabled` | Enable ValidatingWebhookConfiguration | `true` |

## Example Values

### High Availability Setup
```yaml
replicaCount: 3
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 70
```

### Custom Size Limits
```yaml
# 10 GiB limit
config:
  maxPvcSizeBytes: "10737418240"

# 1 TiB limit  
config:
  maxPvcSizeBytes: "1099511627776"
```

### Resource Management
```yaml
resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

## Uninstallation

To uninstall the chart:

```bash
helm uninstall pvc-webhook
```

This will remove all resources associated with the chart except for:
- The namespace (if it existed before installation)
- Any PVCs that were validated by the webhook

## Troubleshooting

### Check installation status:
```bash
helm status pvc-webhook
```

### View webhook logs:
```bash
kubectl logs -l app.kubernetes.io/name=pvc-webhook -n cluster-tools
```

### Check certificate status:
```bash
kubectl get certificates -n cluster-tools
kubectl describe certificate pvc-webhook -n cluster-tools
```

### Verify webhook configuration:
```bash
kubectl get validatingwebhookconfiguration pvc-webhook
kubectl describe validatingwebhookconfiguration pvc-webhook
```