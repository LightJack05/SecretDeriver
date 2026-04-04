# SecretDeriver

SecretDeriver is a Kubernetes operator that **deterministically derives unique secrets** from a single root secret using [HKDF](https://en.wikipedia.org/wiki/HKDF) (HMAC-based Key Derivation Function, SHA-256).

## The problem it solves

Managing secrets in Kubernetes is painful to restore after a cluster rebuild. Approaches like Sealed Secrets or external vaults help, but still require either backups of all sealed secrets, or a running external service. If you lose your secrets, you lose your data.

SecretDeriver takes a different approach: **you only ever need to back up one root secret**. Every other secret in your cluster is derived from it deterministically — recreating the same `DerivedSecret` resource with the same root secret always produces the exact same value, on any cluster, at any time.

## How it works

SecretDeriver introduces a `DerivedSecret` custom resource. Each `DerivedSecret` references a parent Kubernetes `Secret` and a key within it. When reconciled, the operator:

1. Reads the value at `parentSecretKey` from the referenced parent secret
2. Derives a new value using HKDF-SHA256, with `namespace/name` of the `DerivedSecret` as the HKDF salt
3. Writes the derived value into a Kubernetes `Secret` with the same name and namespace as the `DerivedSecret`

Because the salt includes the resource's namespace and name, every `DerivedSecret` produces a **unique** value — even when they share the same root secret and key.

## Key properties

- **Deterministic**: same root secret + same `DerivedSecret` name/namespace = same derived value, always
- **Unique per resource**: different `DerivedSecret` resources always produce different derived values
- **Minimal backup surface**: only the root secret needs to be stored securely
- **Self-healing**: deleted or tampered derived secrets are automatically recreated by the operator
- **Cross-namespace support**: parent secrets in other namespaces can be referenced with an explicit opt-in label

## Getting started

See the [Installing](installing/index.md) section to deploy the operator, then follow the [Getting Started](usage/getting-started.md) guide to create your first derived secret.
