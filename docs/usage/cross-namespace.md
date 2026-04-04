# Cross-Namespace References

By default, a `DerivedSecret` can only reference a parent secret in its own namespace. Referencing a parent secret in a different namespace requires an explicit opt-in on the **parent** secret.

## Enabling cross-namespace access

Add the following label to the parent secret:

```yaml
metadata:
  labels:
    secretderiver.lightjack.de/allowCrossnamespaceReference: "true"
```

Once this label is present, any `DerivedSecret` in any namespace can reference that secret.

## Example

Parent secret in `shared-secrets` namespace:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cluster-root-secret
  namespace: shared-secrets
  labels:
    secretderiver.lightjack.de/allowCrossnamespaceReference: "true"
type: Opaque
stringData:
  root: <your-root-value>
```

`DerivedSecret` in a different namespace:

```yaml
apiVersion: secretderiver.lightjack.de/v1alpha1
kind: DerivedSecret
metadata:
  name: app-secret
  namespace: production
spec:
  parentSecretRef:
    name: cluster-root-secret
    namespace: shared-secrets
  parentSecretKey: root
  generatedSecretKey: secret
```

## Why opt-in?

Without an opt-in mechanism, any `DerivedSecret` in any namespace could silently derive values from secrets belonging to other namespaces. The label requirement ensures that the owner of the parent secret explicitly allows external references, maintaining namespace isolation as a security boundary.

If the label is missing and a cross-namespace reference is attempted, the `DerivedSecret` will enter an error state with reason `ParentSecretNotFound`.
