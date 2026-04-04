# Getting Started

This guide walks through creating your first derived secret.

## 1. Create a root secret

The root secret holds the source value that all derived secrets will be computed from. Use a strong random value — this is the only secret you need to back up.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-root-secret
  namespace: my-namespace
type: Opaque
stringData:
  root-key: <your-strong-random-value>
```

!!! tip
    Generate a root secret value with: `openssl rand -base64 32`

## 2. Create a DerivedSecret

```yaml
apiVersion: secretderiver.lightjack.de/v1alpha1
kind: DerivedSecret
metadata:
  name: my-app-db-password
  namespace: my-namespace
spec:
  parentSecretRef:
    name: my-root-secret
    namespace: my-namespace
  parentSecretKey: root-key
  generatedSecretKey: password
```

Apply it:

```sh
kubectl apply -f derivedsecret.yaml
```

The operator will create a `Secret` named `my-app-db-password` in `my-namespace` containing a `password` key.

## 3. Check the status

```sh
kubectl get derivedsecret my-app-db-password -n my-namespace
```

When ready, `.status.phase` will be `Ready`. See [Status and Conditions](../reference/status.md) for details on other states.

## 4. Use the generated secret in your workloads

```yaml
env:
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: my-app-db-password
        key: password
```

## Multiple derived secrets from the same root

You can create as many `DerivedSecret` resources as you like from the same root secret. Each one produces a **unique** derived value because the `namespace/name` of each `DerivedSecret` is used as the HKDF salt.

```yaml
---
apiVersion: secretderiver.lightjack.de/v1alpha1
kind: DerivedSecret
metadata:
  name: app-db-password
  namespace: production
spec:
  parentSecretRef:
    name: root-secret
    namespace: production
  parentSecretKey: root
  generatedSecretKey: password
---
apiVersion: secretderiver.lightjack.de/v1alpha1
kind: DerivedSecret
metadata:
  name: app-api-token
  namespace: production
spec:
  parentSecretRef:
    name: root-secret
    namespace: production
  parentSecretKey: root
  generatedSecretKey: token
```

Both resources reference the same root key, but `app-db-password` and `app-api-token` will contain different derived values.
