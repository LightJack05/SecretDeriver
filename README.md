# SecretDeriver

A Kubernetes operator that deterministically derives unique secrets from a single root secret using HKDF-SHA256. You only need to back up one root secret — all derived secrets are reproducible on any cluster redeploy.

Each `DerivedSecret` resource points to a parent `Secret` and produces a new `Secret` with a derived value. The derivation uses the `DerivedSecret`'s `namespace/name` as the HKDF salt, so every derived secret is unique but fully reproducible.

**Documentation:** [https://lightjack05.github.io/SecretDeriver/](https://lightjack05.github.io/SecretDeriver/)

## Installation

```sh
helm install secretderiver oci://ghcr.io/lightjack05/charts/secretderiver \
  --namespace secretderiver-system --create-namespace
```

## Usage

```yaml
apiVersion: secretderiver.lightjack.de/v1alpha1
kind: DerivedSecret
metadata:
  name: my-app-secret
  namespace: my-namespace
spec:
  parentSecretRef:
    name: my-root-secret
    namespace: my-namespace
  parentSecretKey: root-key       # key to read from the parent secret
  generatedSecretKey: secret-key  # key to write in the generated secret
```

This creates a `Secret` named `my-app-secret` in `my-namespace` with a `secret-key` entry containing the derived value.

To allow a `DerivedSecret` to reference a parent secret in a different namespace, add this label to the **parent** secret:

```
secretderiver.lightjack.de/allowCrossnamespaceReference: "true"
```

## Status

The `DerivedSecret` reports its state in `.status.phase` (`Ready` or `Error`) and `.status.conditions` with reasons: `DerivationSuccessful`, `ParentSecretNotFound`, `KeyNotFound`, `EmptyValue`, `DerivationFailed`.

## Development

```sh
make install   # install CRDs into current cluster
make run       # run operator locally
make test      # run tests
```
