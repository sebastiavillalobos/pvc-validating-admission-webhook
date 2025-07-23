# PVC Size Validation Webhook

A Kubernetes ValidatingAdmissionWebhook that enforces maximum size limits on PersistentVolumeClaims (PVCs) to prevent users from creating excessively large storage requests.

## What It Does

This webhook intercepts PVC creation and update requests in your Kubernetes cluster and validates them against a configurable maximum size limit. If a PVC requests more storage than the configured limit, the webhook rejects the request with a clear error message.

### Key Features

- **Size Validation**: Validates PVC size requests against a configurable maximum limit
- **Configurable Limits**: Set maximum PVC size via environment variable (default: 5 GiB)
- **Security**: Uses TLS certificates for secure communication with the Kubernetes API server
- **Logging**: Comprehensive logging for monitoring and debugging
- **Cloud Native**: Containerized and ready for Kubernetes deployment

## How It Works

1. **Admission Controller**: The webhook registers as a ValidatingAdmissionWebhook with the Kubernetes API server
2. **Interception**: When a PVC is created or updated, Kubernetes sends the request to the webhook
3. **Validation**: The webhook:
   - Parses the incoming admission review request
   - Extracts the PVC storage request
   - Compares it against the configured maximum size
   - Returns an admission response (allow/deny)
4. **Enforcement**: If the PVC exceeds the limit, Kubernetes rejects the request with the webhook's error message

### Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   kubectl/API  │────│  Kubernetes API  │────│  PVC Webhook    │
│     Client      │    │     Server       │    │   Validator     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │  PVC Creation/   │
                       │     Update       │
                       └──────────────────┘
```

## Configuration

### Environment Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `MAX_PVC_SIZE_BYTES` | Maximum PVC size in bytes | `5368709120` (5 GiB) | `10737418240` (10 GiB) |

### Command Line Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `--port` | Server port | `9093` |
| `--tls-crt` | Path to TLS certificate | `/etc/webhook/certs/tls.crt` |
| `--tls-key` | Path to TLS private key | `/etc/webhook/certs/tls.key` |

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.16+)
- cert-manager installed for TLS certificate management
- kubectl configured to access your cluster

### Using Helm (Recommended)

1. **Install the webhook using Helm:**
   ```bash
   helm install pvc-webhook ./helm-chart/pvc-webhook
   ```

2. **Customize the configuration:**
   ```bash
   helm install pvc-webhook ./helm-chart/pvc-webhook \
     --set config.maxPvcSizeBytes=10737418240 \
     --set replicaCount=2
   ```

### Manual Deployment

1. **Install cert-manager (if not already installed):**
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
   ```

2. **Create the namespace:**
   ```bash
   kubectl create namespace cluster-tools
   ```

3. **Deploy the webhook:**
   ```bash
   kubectl apply -f k8s-files/
   ```

## Testing

### Test Valid PVC (Should Succeed)

Create a PVC within the size limit:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-small-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
EOF
```

### Test Invalid PVC (Should Fail)

Create a PVC exceeding the size limit:

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-large-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
EOF
```

Expected error (with default 5Gi limit):
```
error validating data: ValidationError(PersistentVolumeClaim): PVC size 10Gi exceeds the maximum limit of 5Gi
```

## Development

### Building the Container

```bash
cd pvc-webhook-code
docker build -t your-registry/pvc-webhook:latest .
docker push your-registry/pvc-webhook:latest
```

## License

This project is open source and available under the [MIT License](LICENSE).
