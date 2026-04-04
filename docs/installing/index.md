# Installing SecretDeriver

SecretDeriver is deployed as a Kubernetes operator. The recommended installation method is via the [Helm chart](helm.md).

## Prerequisites

- Kubernetes 1.25+
- Helm 3 (for Helm-based installation)
- Cluster admin permissions to install CRDs and RBAC resources

## What gets installed

The operator deployment includes:

- The `DerivedSecret` CRD
- The controller manager deployment
- A `ServiceAccount` with the necessary RBAC permissions:
    - Cluster-wide read access to `Secrets` (to resolve parent secrets across namespaces)
    - Write access to `Secrets` (to create and update derived secrets)
    - Full access to `DerivedSecret` resources
- A metrics `Service` (port 8443, HTTPS)

!!! note
    The operator requires cluster-wide read access to `Secrets` because parent secrets may be in any namespace. Write access is limited to secrets owned by `DerivedSecret` resources.
