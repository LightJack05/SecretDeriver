# Status and Conditions

The `DerivedSecret` resource reports its state via `.status.phase` and `.status.conditions`.

## Phase

| Phase | Description |
|---|---|
| `Ready` | The derived secret has been successfully created or updated |
| `Error` | An error occurred during reconciliation |

## Conditions

The `.status.conditions` array contains a single condition. The `reason` field describes the specific outcome:

| Reason | Phase | Description |
|---|---|---|
| `DerivationSuccessful` | `Ready` | The secret was derived and written successfully |
| `ParentSecretNotFound` | `Error` | The referenced parent secret does not exist, or cross-namespace access is not enabled on it |
| `KeyNotFound` | `Error` | The specified `parentSecretKey` does not exist in the parent secret |
| `EmptyValue` | `Error` | The specified key in the parent secret has an empty value |
| `DerivationFailed` | `Error` | An internal error occurred during HKDF derivation |

## Recovery behavior

- **`ParentSecretNotFound`**: no automatic requeue. The operator watches all secrets — reconciliation will trigger automatically when the parent secret is created or updated.
- **`KeyNotFound`** and **`EmptyValue`**: requeues every 30 seconds, and also whenever the parent secret changes.
- **`DerivationFailed`**: requeues with exponential backoff.
