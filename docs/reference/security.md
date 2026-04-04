# Security

## Threat model

SecretDeriver is designed around the assumption that you have **one high-value secret** (the root secret) that is stored securely, and that all other secrets in your cluster are derived from it. The security of all derived secrets is therefore entirely dependent on the security of the root secret.

## Derivation security

Derived values are produced using [HKDF](https://en.wikipedia.org/wiki/HKDF) with SHA-256. HKDF is a standard, well-analyzed key derivation function designed specifically for deriving cryptographically strong keys from a shared secret.

- Output is 32 bytes (256 bits), encoded as base64url (43 characters)
- Each `DerivedSecret` uses its `namespace/name` as the HKDF salt, ensuring every derived value is unique even when sharing the same root
- Derived secrets are compared using constant-time comparison (`crypto/subtle`) to avoid timing side channels when deciding whether an update is needed

## RBAC

The operator requires:

- **Cluster-wide read access to `Secrets`** — needed to resolve parent secrets in any namespace
- **Write access to `Secrets`** — limited to creating and updating secrets owned by `DerivedSecret` resources

The Helm chart creates a `ClusterRole` with these permissions and binds it to the operator's `ServiceAccount`.

## Cross-namespace references

Cross-namespace parent references require an explicit opt-in label on the parent secret (`secretderiver.lightjack.de/allowCrossnamespaceReference: "true"`). Without this label, a `DerivedSecret` in namespace A cannot derive from a secret in namespace B, preserving namespace isolation as a security boundary.

See [Cross-Namespace References](../usage/cross-namespace.md) for details.

## Recommendations

- Use a strong, randomly generated root secret (at least 256 bits of entropy). For example: `openssl rand -base64 32`
- Store the root secret securely outside the cluster (e.g. a hardware security module, an offline backup, or a dedicated secrets manager)
- Restrict access to the root secret using Kubernetes RBAC — only the operator and administrators who manage it should be able to read it
- Do not use the same root secret across environments (dev, staging, production) — use separate root secrets per environment
