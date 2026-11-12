# Domain docs

How engineering skills should consume this repository's domain documentation.

## Before exploring, read these

- [`GLOSSARY.md`](../../GLOSSARY.md) at the repository root.
- [`docs/adr/`](../adr/) for decisions that touch the area you are changing.

`GLOSSARY.md` is the only glossary: this is a single-context repository. If a
`GLOSSARY-MAP.md` appears later, treat the repository as multi-context and read
the relevant per-context glossary.

Do not flag missing lazy-created docs. Add ADRs or glossary terms when a real
decision or term is resolved.

## Use the glossary's vocabulary

Use the glossary's terms in issue titles, specs, refactor proposals, test names,
and commit messages. Do not invent synonyms for concepts the glossary already
names.

If a needed concept is missing, either reconsider the language or add the term
through domain modeling.

## Flag ADR conflicts

If a change contradicts an existing ADR, surface it explicitly instead of
silently overriding it. Example:

> Contradicts ADR-0001 (separate CLI and registry repos), but worth reopening
> because…

## Current architecture

- CLI: this repository.
- Registry: `smokeforge-registry`.
- Generated apps: independent projects created by `smokeforge new`.

Read [STACKS.md](../../STACKS.md) for versions and defaults, and
[docs/architecture.md](../architecture.md) for system internals.
