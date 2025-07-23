docker build -t pvc-webhook:1.0.0 .

docker tag pvc-webhook:1.0.0 sebiuo/pvc-webhook:1.0.0

docker push sebiuo/pvc-webhook:1.0.0

## Configuration

The webhook now supports configuring the maximum PVC size via environment variable:
- `MAX_PVC_SIZE_BYTES`: Maximum allowed PVC size in bytes (default: 5368709120 = 5 GiB)