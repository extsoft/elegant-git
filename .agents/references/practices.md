# Engineering practices

Concise reminders for plan and execute skills. Load on demand; do not restate in task files.

## Process

- **Karpathy guardrails:** State assumptions; surface tradeoffs; ask when unclear; minimum code; surgical edits only; verifiable success criteria per step.
- **Spec-driven:** Behavior intent → plan → tasks → code → tests; keep traceability from task to plan anchor.
- **ADRs:** Non-trivial decisions get ADR-lite in `plan.md` (decision, rejected alternatives, consequences).
- **Trunk-based + small PRs:** One task ≈ one PR; merge frequently; avoid long-lived branches.
- **Definition of Done:** Every task must satisfy [definition-of-done.md](definition-of-done.md) plus its `acceptance` list.

## Design and code quality

- **TDD:** Write failing test first when behavior changes; make it pass; refactor.
- **YAGNI / DRY / KISS:** No speculative features; one abstraction per real duplication; prefer obvious solutions.
- **Domain language:** Same terms in plan, tasks, code, and user-facing docs.
- **Bounded cleanup:** Remove orphans your change created; do not widen scope to fix unrelated code.

## Security and reliability

- **Security:** Validate inputs; least privilege; no secrets in repo; review auth and data paths.
- **Threat model:** For security-sensitive changes, note assets, threats, and mitigations in `plan.md`.
- **Supply-chain:** Pin dependencies; commit lockfiles; review new packages before adding.
- **12-factor (strict):** Config via environment only; no secrets in source; stateless processes where applicable.
- **Rollback:** Document how to revert or disable the change in `plan.md`; prefer feature flags for risky paths.
- **Data migration:** Plan up/down or backfill steps in dedicated tasks; never mix schema change with unrelated edits.

## Operations

- **Observability:** Structured logs; no PII in logs; meaningful error messages without internal leakage.
- **Performance budgets:** Note expected complexity for hot paths; add regression check when perf matters.
- **Deprecation log:** Contract removals or breaking API changes update the canonical deprecation log in the same change as code.

## Delivery

- **Docs-as-code:** Update README, man pages, or `docs/` in the same change as behavior.
- **Lint + format:** From repo root, run **`mise run fix`** before marking a task done (not `mise run check`—that is for CI). Use **`mise run test`** for Go tests when applicable. See [`.cursor/rules/mise.mdc`](../../.cursor/rules/mise.mdc).
- **CI green:** Do not merge on red; document flake retries if known.
- **Code-review self-check:** Diff sanity, error paths, naming, tests cover acceptance criteria.
- **Git hygiene:** Atomic commits per task; clean history (rebase when appropriate); signed commits when project requires.
