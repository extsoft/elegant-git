---
name: ubiquitous-language
description: >
  Extract and formalize domain terminology into .agents/references/glossary.md.
  Use when the user asks to define terms, build a glossary, align vocabulary,
  resolve ambiguous/synonymous wording, or capture domain language from the
  current conversation—even without saying "ubiquitous language."
disable-model-invocation: true
---

# ubiquitous-language

## Procedure

1. Scan the conversation for domain nouns, verbs, and concepts.
2. Flag ambiguity (same word, different concepts), synonyms (different words, same concept), and vague or overloaded terms.
3. Propose canonical term choices.
4. Read existing `.agents/references/glossary.md` if present; merge new or updated entries.
5. Write `.agents/references/glossary.md`.
6. Summarize changes inline in the conversation.

## Output format

File starts with `# Glossary` and alphabetically-sorted `### Term` blocks. No tables.

```markdown
### Term
Short canonical definition (one sentence).

Aliases: alt1, alt2

Avoid: deprecated/wrong terms (why)

Notes: scope, edge cases (optional)
```

## Rules

- Keep entries short; one canonical term per concept.
- Preserve existing entries unless the user approves changes.
- Never silently rename established terms.

## Gotchas

- Merge duplicates—do not append a second entry for the same concept.
- Call out conflicts with existing entries before overwriting.
- Cite source (file/line or message) only when it disambiguates.
