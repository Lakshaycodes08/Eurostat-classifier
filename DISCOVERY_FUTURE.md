# Swytchcode Discovery — Future Roadmap

Items deferred from the current discovery sprint. Revisit after Discovery v1 and Third-Party integration are live.

---

## Phase: AI-Suggested Workflow Composition

**When:** After verified workflow library is large enough to validate AI output against.

- [ ] Add `swytchcode discover` fallback: if no verified workflow matches, fetch top-N methods from Weaviate
- [ ] Use gpt-4o to sequence methods into a suggested workflow based on intent
- [ ] Return suggested workflows clearly flagged as `⚠ AI-suggested, not verified`
- [ ] Block `swytchcode exec` on AI-suggested workflows until saved
- [ ] Add `swytchcode workflow save <name>` — promotes a suggested workflow to verified, stores in MySQL + re-indexes Weaviate
- [ ] Output format:
  ```
  Suggested Workflow (AI-composed — review before use)
  ─────────────────────────────────────────────────────
  Steps: customer.fetch → payment.charge → email.receipt.send

  Save it:  swytchcode workflow save "order.charge_and_notify"
  ```

---

## Phase: Workflow CLI Management

**When:** After multi-library execution (THIRD_PARTY_TODO) is stable.

- [ ] `swytchcode workflow create` — interactive or file-based workflow definition locally
  - Generates a YAML/JSON workflow spec
  - Validates that all referenced canonical_ids exist in the registry
- [ ] `swytchcode workflow publish` — submits local workflow to backend for review/indexing
  - Validates steps, checks method existence, generates canonical_id
  - Triggers Weaviate re-index on publish
- [ ] `swytchcode workflow list` — lists all verified workflows for a project
- [ ] `swytchcode workflow delete <canonical_id>` — soft-delete, marks `is_active = false`

---

## Phase: Intent Phrases Layer

**When:** After discovery quality feedback shows semantic misses.

- [ ] Add `intent_phrases` JSON column to `methods` and `workflows` tables
  - Example: `["charge customer", "bill user", "process payment", "collect subscription"]`
- [ ] Include `intent_phrases` in `embed_text` during Weaviate indexing
- [ ] Expose intent phrase editing in the app UI (for integration owners to tune)
- [ ] Write migration script to seed initial intent phrases using gpt-4o-mini from existing descriptions

---

## Phase: Embedding Model Upgrade

**When:** Discovery misses are consistently reported after v1 launch.

- [ ] Evaluate `text-embedding-3-large` vs current `text-embedding-3-small`
  - ~20% better retrieval accuracy, ~5x cost
  - Requires full Weaviate re-index across all namespaces
- [ ] Consider Gemini `text-embedding-004` as cost-competitive alternative
- [ ] Run A/B comparison on a sample of real discovery queries before committing

---

## Phase: Hosted Discovery API (for external agents)

**When:** There is demand from external AI agent builders.

- [ ] Expose a public `POST /v1/discover` endpoint (separate from CLI route)
- [ ] API key auth for external consumers
- [ ] Rate limiting per key
- [ ] LLM-optimized response format (compact JSON, no UI hints)
- [ ] Publish OpenAPI spec so agents can auto-discover the discovery API
