// policy.go will hold execution policy (retries, idempotency, guardrails); all behavior lives in the kernel, not editors or thin clients.
package kernel

// This file will contain policy logic applied during execution, such as:
//
//   - Idempotency handling (e.g. keys, safe retries).
//   - Retry strategies and limits for network/transient failures.
//   - Any guardrails required by Wrekenfile or tooling.json contracts.
//
// The key principle is that all such behavior lives in the kernel and
// not in editors or thin clients.

