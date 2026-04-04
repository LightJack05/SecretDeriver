# DerivedSecret Reference

## API version

`secretderiver.lightjack.de/v1alpha1`

## Example

```yaml
apiVersion: secretderiver.lightjack.de/v1alpha1
kind: DerivedSecret
metadata:
  name: my-secret
  namespace: my-namespace
spec:
  parentSecretRef:
    name: my-root-secret
    namespace: my-namespace
  parentSecretKey: root-key
  generatedSecretKey: secret-value
```

## Spec fields

| Field | Type | Required | Description |
|---|---|---|---|
| `parentSecretRef.name` | string | yes | Name of the parent `Secret` |
| `parentSecretRef.namespace` | string | yes | Namespace of the parent `Secret` |
| `parentSecretKey` | string | yes | Key within the parent secret to use as the derivation input |
| `generatedSecretKey` | string | yes | Key name to write in the generated `Secret` |

## Generated secret

The operator creates a `Secret` with:

- **Name**: same as the `DerivedSecret`
- **Namespace**: same as the `DerivedSecret`
- **Owner reference**: set to the `DerivedSecret`, so it is garbage-collected when the `DerivedSecret` is deleted
- **Data**: a single key named `generatedSecretKey` containing the derived value

The derived value is 32 bytes of HKDF-SHA256 output, encoded as base64url (43 characters, no padding).

## Derivation details

The derivation uses the following inputs:

| HKDF parameter | Value |
|---|---|
| Hash | SHA-256 |
| Input key material (IKM) | Value of `parentSecretKey` in the parent secret |
| Salt | `<namespace>/<name>` of the `DerivedSecret` |
| Info | _(empty)_ |
| Output length | 32 bytes |

The salt ensures that each `DerivedSecret` produces a unique value, even when multiple resources reference the same root key.
