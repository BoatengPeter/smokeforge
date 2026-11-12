# Issue tracker: Local Markdown

Issues and specs for this repository live as markdown files under `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`
- Spec: `.scratch/<feature-slug>/spec.md`
- Implementation tickets: `.scratch/<feature-slug>/issues/<NN>-<slug>.md`,
  numbered from `01`, one file per ticket
- Triage state is a `Status:` line near the top of each issue file
- Conversation history appends under `## Comments`

## When a skill says "publish to the issue tracker"

Create a new file under `.scratch/<feature-slug>/`, creating directories as
needed.

## When a skill says "fetch the relevant ticket"

Read the referenced file. The user normally passes the path or issue number.

## Wayfinding

- Map: `.scratch/<effort>/map.md`
- Child ticket: `.scratch/<effort>/issues/NN-<slug>.md`
- `Type:` records `research`, `prototype`, `grilling`, or `task`
- `Status:` records `claimed` or `resolved`
- `Blocked by: NN, NN` lists unresolved dependencies
- Claim by setting `Status: claimed` before work
- Resolve by appending `## Answer`, setting `Status: resolved`, and recording the
  decision in `map.md`

## Scope

This tracker is for smokeforge engineering work. Generated client projects are
separate repositories with their own trackers unless the user says otherwise.
