Kubernetes manifests for the `L8PKubernetesAPI` admission collector live here.

`validating-webhook.yaml` is generated from the active Pollaris boot polls. To regenerate it:

```bash
cd go
GOCACHE=/tmp/go-build go run ./cmd/k8s-webhook-config \
  -name l8collector-k8s \
  -service l8collector-admission \
  -namespace default \
  -path /admission/kubernetes
```

The current checked-in manifest assumes:

- Service name: `l8collector-admission`
- Namespace: `default`
- Admission path: `/admission/kubernetes`
- ClusterName: `lab`

`admission-control.yaml` deploys the `go/adcon` image and exposes HTTPS on port `8443`.
It also creates the service account, RBAC, TLS bootstrap job, and the deployment.

The bootstrap job:

- generates a self-signed certificate for `l8collector-admission`, `l8collector-admission.default`, and `l8collector-admission.default.svc`
- creates or updates the `l8collector-admission-tls` secret
- patches the `ValidatingWebhookConfiguration` with the matching `caBundle`

The deployment mounts that secret as:

- `/data/admission.crt`
- `/data/admission.crtKey`

Apply order matters because the bootstrap job patches the webhook object:

```bash
kubectl apply -f k8s/validating-webhook.yaml
kubectl apply -f k8s/admission-control.yaml
```

Wrapper scripts are included:

```bash
./k8s/deploy-adm.sh
./k8s/un-deploy-adm.sh
```
