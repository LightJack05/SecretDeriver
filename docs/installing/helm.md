# Installing with Helm

The SecretDeriver Helm chart is published to the GitHub Container Registry as an OCI chart.

## Install

```sh
helm install secretderiver oci://ghcr.io/lightjack05/charts/secretderiver \
  --namespace secretderiver-system \
  --create-namespace
```

## Upgrade

```sh
helm upgrade secretderiver oci://ghcr.io/lightjack05/charts/secretderiver \
  --namespace secretderiver-system
```

## Uninstall

```sh
helm uninstall secretderiver --namespace secretderiver-system
```

!!! warning
    Uninstalling the chart removes the operator and its RBAC resources, but does **not** delete existing `DerivedSecret` resources or the secrets they created. Clean those up manually if needed.

## Configuration

The following values can be overridden with `--set` or a values file:

| Value | Default | Description |
|---|---|---|
| `controllerManager.manager.image.repository` | `ghcr.io/lightjack05/secretderiver` | Container image repository |
| `controllerManager.manager.image.tag` | `""` (chart appVersion) | Container image tag |
| `controllerManager.replicas` | `1` | Number of operator replicas |
| `controllerManager.manager.resources.limits.cpu` | `500m` | CPU limit |
| `controllerManager.manager.resources.limits.memory` | `128Mi` | Memory limit |
| `controllerManager.nodeSelector` | `{}` | Node selector for the operator pod |
| `controllerManager.tolerations` | `[]` | Tolerations for the operator pod |
| `controllerManager.topologySpreadConstraints` | `[]` | Topology spread constraints |
| `serviceAccount.create` | `true` | Whether to create a ServiceAccount |
| `serviceAccount.name` | `""` | ServiceAccount name (auto-generated if empty) |
| `serviceAccount.annotations` | `{}` | Annotations to add to the ServiceAccount |

Example — pinning to a specific image tag:

```sh
helm install secretderiver oci://ghcr.io/lightjack05/charts/secretderiver \
  --namespace secretderiver-system \
  --create-namespace \
  --set controllerManager.manager.image.tag=v1.0.0
```
