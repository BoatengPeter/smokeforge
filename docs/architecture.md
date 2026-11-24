# smokeforge architecture

Visual companion to the prose architecture docs. Diagrams distinguish
**implemented** infrastructure from **planned** generation behavior, matching the
[status legend](README.md#status-legend).

Related reading:

- [System overview](architecture/overview.md)
- [CLI and lifecycle](architecture/cli.md)
- [Generation engine](architecture/generation-engine.md)
- [Registry manifest](registry/manifest.md)
- [Overlay model](registry/overlays.md)

---

## 1. System context

```mermaid
flowchart LR
    user["Developer or CI"] -->|"wizard or flags"| cli["smokeforge CLI"]
    cli -->|"fetch tag, tarball, checksum"| registry[("smokeforge-registry\nGitHub releases")]
    cli -->|"render, patch, stage"| project["Generated project"]
    cli -->|"git init"| git["git"]
    cli -->|"smokeforge install"| tools["Toolchains\nGo, Cargo, uv, npm,\nGradle, Xcode"]
    registry -.->|"registry.tar.gz\nregistry.sha256"| cli
    project -.->|".smokeforge.json\n.smokeforge.lock.json"| cli
```

Trust boundary: the registry is remote and untrusted. Everything it contains is
declarative; the CLI decides what is written and never executes template code.

---

## 2. Component map

```mermaid
flowchart TD
    cmd["cmd/smokeforge"] --> cli["internal/cli\nCobra commands"]
    cli --> tui["internal/tui\ninteractive wizard"]

    cli --> resolve["internal/resolve\nselection to plan"]
    cli --> registry["internal/registry\nsources, cache, checksums"]
    registry --> manifest["internal/manifest\nschema and validation"]

    resolve --> engine["internal/engine\npreflight, stage, commit"]
    engine --> render["internal/render\nstrict templates"]
    engine --> patch["internal/patch\nanchors and structured ops"]
    engine --> project["internal/project\nstate, lockfile, hashes"]
    engine --> fsx["internal/fsx\natomic writes, path safety"]
    engine --> gitx["internal/gitx\ngit init"]

    cli --> install["internal/install\nper-stack install steps"]
    project --> resolve
    install --> toolchains["Go, Rust, Python,\nNode, Android, iOS"]

    subgraph implemented["Implemented"]
        resolve
        engine
        render
        patch
        project
        fsx
        gitx
        install
        tui
    end
```

Dependency rule: `cli` depends on libraries; libraries never import `cli`.
`engine` is the only package that mutates targets. `manifest` never performs
network I/O.

---

## 3. New project sequence

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant W as Wizard / flags
    participant R as Resolver
    participant G as Registry
    participant E as Engine
    participant P as Project state
    participant F as Filesystem

    U->>W: kind, stacks, features, metadata
    W->>R: normalized selection
    R->>G: load manifest (cache first)
    G-->>R: validated manifest
    R->>R: enforce kinds, variants, requires/conflicts
    R-->>E: ordered generation plan
    E->>E: preflight anchors and patch paths
    E->>F: render into sibling staging dir
    E->>E: apply base templates then overlays
    E->>P: write .smokeforge.json + lockfile
    E->>F: write .env, git init, atomic rename
    F-->>U: project ready, next steps printed
    Note over E,F: Any failure before rename leaves the target untouched
```

---

## 4. Registry resolution state machine

```mermaid
stateDiagram-v2
    [*] --> SourceCheck
    SourceCheck --> LocalLoad: source is a directory
    SourceCheck --> TagCheck: source is HTTP(S)
    LocalLoad --> ManifestValidate
    ManifestValidate --> Ready
    ManifestValidate --> Failed: schema or reference error

    TagCheck --> Failed: invalid tag
    TagCheck --> SourceSecurity
    SourceSecurity --> Failed: non-TLS non-loopback host
    SourceSecurity --> CacheLookup
    CacheLookup --> ManifestValidate: cache hit
    CacheLookup --> OfflineError: miss and --offline
    CacheLookup --> Download: miss and online
    Download --> ChecksumVerify
    ChecksumVerify --> Failed: checksum mismatch
    ChecksumVerify --> SafeExtract
    SafeExtract --> Failed: traversal, link, limit
    SafeExtract --> CacheCommit
    CacheCommit --> ManifestValidate
    ManifestValidate --> Ready

    OfflineError --> [*]
    Failed --> [*]
    Ready --> [*]
```

Cache entry key: `sha256(source + NUL + tag)` truncated to 16 hex characters.
Cache commits are atomic: download to a temp directory, verify, extract, then
rename into place.

---

## 5. Overlay composition

```mermaid
flowchart LR
    base["Base template\ngo/gin-layered, rust, python,\nnext, swiftui, compose,\ngpui, egui"] --> render["Strict render\nvariables + secrets"]
    render --> anchors["Anchored file tree\nsmokeforge:anchor imports\nsmokeforge:anchor routes"]

    subgraph overlays["Feature overlays (dependency ordered)"]
        db["database:postgres"]
        cache["cache:redis"]
        queue["queue:rabbitmq"]
        auth["auth:jwt"]
        totp["totp\nrequires auth:jwt"]
        infra["docker + ci:github"]
    end

    anchors --> ops["Declarative operations\naddFiles\ninsertBefore/After\npatchJSON/YAML/TOML\naddDependency\nenv"]
    overlays --> ops
    ops --> tree["Generated project"]

    subgraph markers["Reversible insertion"]
        begin["smokeforge:begin cache:redis"] --> code["generated block"] --> end["smokeforge:end cache:redis"]
    end
    ops -.-> markers
```

Every anchor must match exactly once. Insertions are idempotent and wrapped in
markers; structured patches are re-parsed after editing; dependency edits use
language-aware adapters.

---

## 6. Project data model

```mermaid
erDiagram
    PROJECT_STATE ||--|| REGISTRY_LOCK : pins
    PROJECT_STATE ||--o{ TARGET : contains
    TARGET ||--o{ APPLIED_OVERLAY : records
    APPLIED_OVERLAY ||--o{ FILE_HASH : tracks
    REGISTRY_LOCK ||--o{ TEMPLATE_CHECKSUM : verifies
    REGISTRY_LOCK ||--o{ OVERLAY_CHECKSUM : verifies

    PROJECT_STATE {
        int schemaVersion
        string name
        string kind
        string createdAt
        json variables
        json rootFeatures
    }
    TARGET {
        string path
        string stack
        json features
    }
    APPLIED_OVERLAY {
        string overlay
        string target
        string registryTag
        json markers
    }
    FILE_HASH {
        string path
        string sha256
    }
    REGISTRY_LOCK {
        string source
        string tag
        string commit
        string archiveSHA256
    }
    TEMPLATE_CHECKSUM {
        string template
        string sha256
    }
    OVERLAY_CHECKSUM {
        string overlay
        string sha256
    }
```

`FILE_HASH` records post-generation content so `add`/`remove` can detect drift
without touching user edits.

---

## 7. Fullstack composition

```mermaid
flowchart TD
    wizard["fullstack selection"] --> api["apps/api\nGo or Rust"]
    wizard --> web["apps/web\nNext.js (explicit --frontend next)"]
    wizard --> ios["apps/ios\nSwiftUI"]
    wizard --> android["apps/android\nCompose"]
    wizard --> desktop["apps/desktop\nElectron (standalone implemented)"]
    wizard --> root["workspace/fullstack, workspace/mobile, or workspace/monorepo"]

    api --> apiFeatures["database, cache, queue,\nauth, totp"]
    root --> repoFeatures["docker, ci"]
    repoFeatures --> compose["compose.yaml"]
    repoFeatures --> workflows[".github/workflows/ci.yml"]
    api --> wiring["API base URL and BFF wiring"]
    wiring --> web
    wiring --> ios
    wiring --> android
    wiring -.-> desktop
```

Root overlays aggregate services and workflows from target metadata. Overlays
may not write outside their declared target.

---

## 8. Mutation transaction (`add` / `remove`)

```mermaid
sequenceDiagram
    autonumber
    actor U as User
    participant C as CLI
    participant S as Project state
    participant F as Filesystem
    participant B as Backups

    U->>C: smokeforge add cache:redis
    C->>S: load state + lockfile
    C->>C: resolve overlay against locked registry
    C->>F: hash impacted files
    C->>C: compare with recorded hashes
    alt drift detected
        C-->>U: show summary, refuse or ask
    end
    C->>B: snapshot impacted files
    C->>F: write transaction marker
    C->>F: apply verified operations
    alt failure during apply
        C->>B: restore snapshot
        C->>F: clear transaction marker
        C-->>U: error, target unchanged
    else success
        C->>S: update hashes and lockfile
        C->>F: clear transaction marker
        C-->>U: overlay applied
    end
```

Interrupted mutations leave a transaction marker and backup so `doctor` can
report recovery steps.

---

## 9. Trust boundaries

```mermaid
flowchart LR
    subgraph untrusted["Untrusted input"]
        reg["Registry archive"]
        tag["Registry tag"]
        src["Registry URL"]
    end

    subgraph checks["CLI checks"]
        tagCheck["Tag regex\nletters, digits, . _ -"]
        tls["HTTPS required\nHTTP only for loopback"]
        sum["SHA-256 verification"]
        extract["os.Root extraction\nno links, no traversal"]
        limits["Entry, file, total limits"]
        schema["Strict manifest schema\nunknown fields rejected"]
        anchors["Anchor matched exactly once"]
    end

    subgraph safe["Trusted side effects"]
        stage["Staging directory"]
        write["Generated files"]
        git["git init"]
        install["Install adapters\nbuilt-in commands only"]
    end

    reg --> sum --> extract --> limits --> schema --> stage --> write --> git
    tag --> tagCheck --> stage
    src --> tls --> stage
    write -.-> anchors
    write --> install
```

Templates cannot execute shell commands. `smokeforge install` uses built-in,
per-stack adapters rather than command strings from the registry.

---

## 10. Testing and CI

```mermaid
flowchart TD
    unit["Unit tests\nmanifest, registry, patch,\nresolve, engine, project"] --> contract["Registry contract tests\nevery overlay x supported base"]
    contract --> golden["Golden trees\ndeterministic output"]
    golden --> build["Native build matrix\nGo, Cargo, uv, npm,\nGradle, xcodebuild"]
    build --> e2e["CLI end-to-end\nnew / add / remove / install"]
    e2e --> release["goreleaser + registry release"]

    unit --> ci["CLI CI\ngo test ./...\ngolangci-lint"]
    contract --> registryCI["Registry CI"]
    build --> registryCI
```

Full testing details: [testing strategy](development/testing.md).

---

## Status map

| Diagram area | Status |
| --- | --- |
| CLI entrypoint, version, list, registry, doctor | Implemented |
| Manifest validation | Implemented |
| Local and remote registry loading, checksums, cache, safe extraction | Implemented |
| Resolver, renderer, patch engine, staging, state/lock, wizard | Implemented (M2) |
| Base templates and feature overlays | Implemented (M3: API + Next.js) |
| Add/remove lifecycle, drift, backups, rollback | Implemented (M4 lifecycle) |
| Install | Planned (M4 remainder) |
| Release pipeline | Implemented (M5) |
