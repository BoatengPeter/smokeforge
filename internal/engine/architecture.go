package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
	"github.com/BoatengPeter/smokeforge/internal/resolve"
)

const (
	architectureFile = "ARCHITECTURE.md"
	kindBackend      = "backend"
	kindDesktop      = "desktop"
	kindFrontend     = "frontend"
	kindFullstack    = "fullstack"
	kindMobile       = "mobile"
	iosStackID       = "ios/swiftui"
)

type capability struct {
	name string
	deps []string
}

type dependency struct {
	module  string
	version string
}

func writeArchitecture(root string, plan *resolve.Plan, m *manifest.Manifest) error {
	content, err := buildArchitecture(root, plan, m)
	if err != nil {
		return err
	}

	//nolint:gosec // generated project documentation is intentionally readable.
	if err := os.WriteFile(filepath.Join(root, architectureFile), []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", architectureFile, err)
	}

	return nil
}

func buildArchitecture(root string, plan *resolve.Plan, m *manifest.Manifest) (string, error) {
	capabilities, dependencies := architectureCapabilities(plan, m)

	var b strings.Builder
	b.WriteString("# Architecture\n\n")
	b.WriteString("This file describes the project's app boundaries, runtime flow, and where to make common changes.\n\n")

	writeSystemDiagram(&b, plan)
	writeDirectoryMap(&b, root, plan)
	writeChangeMap(&b, plan)

	if len(capabilities) > 0 {
		b.WriteString("## Included capabilities\n\n")

		for _, capability := range capabilities {
			b.WriteString("- ")
			b.WriteString(capability.name)
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	if len(dependencies) > 0 {
		b.WriteString("## Dependencies added by capabilities\n\n")
		b.WriteString("| Package / module | Version |\n| --- | --- |\n")

		for _, dependency := range dependencies {
			fmt.Fprintf(&b, "| `%s` | `%s` |\n", dependency.module, dependency.version)
		}

		b.WriteString("\n")
	}

	b.WriteString("## More detail\n\n")
	b.WriteString("- `CONTEXT.md` — engineering boundaries and non-negotiables.\n")
	b.WriteString("- `docs/` — product, API, storage, UI, and packaging notes.\n")
	b.WriteString("- Each app's `README.md` and `CONTEXT.md` document its module layout and local commands.\n")

	return b.String(), nil
}

func architectureCapabilities(plan *resolve.Plan, m *manifest.Manifest) ([]capability, []dependency) {
	byName := make(map[string]map[string]bool)
	dependencySet := make(map[string]bool)

	for _, target := range allPlanTargets(plan) {
		for _, overlay := range target.Overlays {
			name := capabilityName(m, overlay)
			if name == "" {
				continue
			}

			if byName[name] == nil {
				byName[name] = make(map[string]bool)
			}

			for _, op := range overlay.Overlay.Ops {
				if op.AddDependency == nil {
					continue
				}

				label := op.AddDependency.Module + "|" + op.AddDependency.Version
				if dependencySet[label] {
					continue
				}

				dependencySet[label] = true
				byName[name][label] = true
			}
		}
	}

	capabilities := make([]capability, 0, len(byName))
	for name, deps := range byName {
		items := make([]string, 0, len(deps))
		for dep := range deps {
			items = append(items, dep)
		}

		sort.Strings(items)
		capabilities = append(capabilities, capability{name: name, deps: items})
	}

	sort.Slice(capabilities, func(i, j int) bool { return capabilities[i].name < capabilities[j].name })

	dependencies := make([]dependency, 0, len(dependencySet))
	for item := range dependencySet {
		parts := strings.SplitN(item, "|", 2)
		dependencies = append(dependencies, dependency{module: parts[0], version: parts[1]})
	}

	sort.Slice(dependencies, func(i, j int) bool {
		if dependencies[i].module == dependencies[j].module {
			return dependencies[i].version < dependencies[j].version
		}

		return dependencies[i].module < dependencies[j].module
	})

	return capabilities, dependencies
}

func baseCapabilityName(name string) string {
	if before, _, ok := strings.Cut(name, " — "); ok {
		return before
	}

	return name
}

func capabilityName(m *manifest.Manifest, overlay resolve.OverlayPlan) string {
	for _, feature := range m.Features {
		if feature.ID != overlay.FeatureID {
			continue
		}

		if overlay.VariantID == "" {
			return feature.Name
		}

		for _, variant := range feature.Variants {
			if variant.ID == overlay.VariantID && variant.Name != "" {
				return baseCapabilityName(feature.Name) + " — " + variant.Name
			}
		}

		return baseCapabilityName(feature.Name)
	}

	return ""
}

//nolint:gocyclo // The diagram has one branch per supported target kind.
func writeSystemDiagram(b *strings.Builder, plan *resolve.Plan) {
	paths := targetPaths(plan)
	deps := dependencyNodes(plan)

	b.WriteString("## System\n\n```mermaid\nflowchart LR\n")

	switch plan.Kind {
	case kindFullstack:
		if slices.Contains(paths, "apps/web") {
			b.WriteString("    Browser[Browser] -->|HttpOnly session| Web[Next.js web + BFF]\n")
			b.WriteString("    Web -->|Bearer token| API[API service]\n")
		} else {
			b.WriteString("    Client[API client] --> API[API service]\n")
		}

		b.WriteString("    API --> DB[(PostgreSQL)]\n")

		if slices.Contains(paths, "apps/android") {
			b.WriteString("    Android[Android app] -->|authenticated API| API\n")
		}

		if slices.Contains(paths, "apps/ios") {
			b.WriteString("    iOS[iOS app] -->|authenticated API| API\n")
		}

		if slices.Contains(paths, "apps/desktop") {
			b.WriteString("    Desktop[Desktop app] --> Local[(SQLite)]\n")
			b.WriteString("    Desktop -->|sync| API\n")
		}
	case kindMobile:
		switch {
		case slices.Contains(paths, "apps/api"):
			if slices.Contains(paths, "apps/android") {
				b.WriteString("    Android[Android app] -->|authenticated API| API[API service]\n")
			}

			if slices.Contains(paths, "apps/ios") {
				b.WriteString("    iOS[iOS app] -->|authenticated API| API[API service]\n")
			}

			b.WriteString("    API --> DB[(PostgreSQL)]\n")
		case targetStackKind(plan, kindMobile) == iosStackID:
			b.WriteString("    iOS[iOS app]\n")
		default:
			b.WriteString("    Android[Android app]\n")
		}
	case kindDesktop:
		if desktopStack(plan) == "desktop/electron" {
			b.WriteString("    Renderer[React renderer] -->|typed IPC| Main[Electron main]\n")
			b.WriteString("    Main --> Local[(SQLite)]\n")

			if hasFeature(plan, "sync") || hasFeature(plan, "auth") {
				b.WriteString("    Main -->|authenticated API| API[API service]\n")
			}
		} else {
			b.WriteString("    UI[Rust desktop UI] --> Services[App services]\n")
			b.WriteString("    Services --> Local[(SQLite)]\n")

			if hasFeature(plan, "sync") || hasFeature(plan, "auth") {
				b.WriteString("    Services -->|authenticated API| API[API service]\n")
			}
		}
	case kindBackend:
		b.WriteString("    Client[API client] --> API[API service]\n")
		b.WriteString("    API --> DB[(PostgreSQL)]\n")
	default:
		b.WriteString("    Browser[Browser] --> Web[Next.js app]\n")

		if hasFeature(plan, "auth") || hasFeature(plan, "data") {
			b.WriteString("    Web -->|server-side API| API[API service]\n")
		}
	}

	for _, node := range deps {
		b.WriteString(node)
	}

	b.WriteString("```\n\n")
}

func dependencyNodes(plan *resolve.Plan) []string {
	nodes := make([]string, 0, 4)
	if hasFeature(plan, "cache") {
		nodes = append(nodes, "    API --> Cache[(Redis)]\n")
	}

	if hasFeature(plan, "queue") || hasFeature(plan, "search-sync") {
		nodes = append(nodes, "    API --> Queue[(RabbitMQ)]\n")
	}

	if hasFeature(plan, "search") {
		nodes = append(nodes, "    API --> Search[(Meilisearch)]\n")
	}

	if hasFeature(plan, "media") {
		nodes = append(nodes, "    API --> Media[(Object storage)]\n")
	}

	return nodes
}

func writeDirectoryMap(b *strings.Builder, root string, plan *resolve.Plan) {
	b.WriteString("## Project map\n\n```text\n")

	targets := allPlanTargets(plan)
	for _, target := range targets {
		if target.Path != "." {
			continue
		}

		b.WriteString("./\n")

		for _, entry := range layoutEntries(root, target) {
			fmt.Fprintf(b, "  %s\n", entry)
		}
	}

	for _, target := range targets {
		if target.Path == "." {
			continue
		}

		fmt.Fprintf(b, "%s/\n", target.Path)

		for _, entry := range layoutEntries(root, target) {
			fmt.Fprintf(b, "  %s\n", entry)
		}
	}

	b.WriteString("```\n\n")
}

func layoutEntries(root string, target resolve.TargetPlan) []string {
	candidates := []string{"README.md", "CONTEXT.md", "AGENTS.md", "CLAUDE.md", "docs/"}

	switch target.Stack.Kind {
	case kindFullstack:
		candidates = append(candidates, "CONTEXT-MAP.md", "Makefile", "compose.yaml", ".github/")
	case kindBackend:
		candidates = append(candidates, "cmd/", "internal/", "src/", "migrations/", "tests/")
	case kindFrontend:
		candidates = append(candidates, "src/app/", "src/components/", "src/lib/", "public/")
	case kindMobile:
		if target.Stack.ID == iosStackID {
			candidates = append(candidates, "project.yml", "Makefile", "App/", "Packages/", "Config/", "DESIGN.md", "skills-lock.json")
		} else {
			candidates = append(candidates, "app/", "core/", "feature/", "build-logic/", "gradle/")
		}
	case kindDesktop:
		if target.Stack.ID == "desktop/electron" {
			candidates = append(candidates, "src/main/", "src/preload/", "src/renderer/", "tests/", "assets/")
		} else {
			candidates = append(candidates, "crates/core/", "crates/storage/", "crates/api-client/", "crates/app/", "crates/desktop/", "assets/")
		}
	}

	entries := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		path := filepath.Join(root, filepath.FromSlash(target.Path), filepath.FromSlash(strings.TrimSuffix(candidate, "/")))
		if _, err := os.Stat(path); err == nil {
			entries = append(entries, candidate)
		}
	}

	return entries
}

func writeChangeMap(b *strings.Builder, plan *resolve.Plan) {
	b.WriteString("## Where to change what\n\n")
	b.WriteString("| Area | Start here |\n| --- | --- |\n")

	for _, target := range allPlanTargets(plan) {
		if target.Path == "." {
			writeSingleAppRows(b, target)
			continue
		}

		writeTargetRows(b, target)
	}

	b.WriteString("\n")
}

func writeSingleAppRows(b *strings.Builder, target resolve.TargetPlan) {
	switch target.Stack.Kind {
	case kindBackend:
		b.WriteString("| HTTP/API behavior | `README.md`, `CONTEXT.md`, `docs/API.md` |\n")
		b.WriteString("| Domain and persistence | `CONTEXT.md`, `migrations/`, `internal/` or `src/` |\n")
	case kindFrontend:
		b.WriteString("| Pages and layouts | `src/app/`, `src/components/`, `README.md` |\n")
		b.WriteString("| Data fetching and BFF | `src/lib/`, `src/app/api/`, `CONTEXT.md` |\n")
	case kindMobile:
		if target.Stack.ID == iosStackID {
			b.WriteString("| Screens and navigation | `App/`, `Packages/*Feature/`, `DESIGN.md` |\n")
			b.WriteString("| Auth and network | `Packages/Networking/`, `Packages/Data/`, `docs/AUTH.md`, `docs/NETWORK.md` |\n")
			b.WriteString("| Secure token storage | `Packages/Storage/`, `docs/STORAGE.md` |\n")
			b.WriteString("| Xcode project and configs | `project.yml`, `Config/`, `Makefile` |\n")
		} else {
			b.WriteString("| Screens and navigation | `app/`, `feature/`, `docs/ARCHITECTURE.md` |\n")
			b.WriteString("| Auth and network | `core/network/`, `core/data/`, `docs/AUTH.md`, `docs/NETWORK.md` |\n")
			b.WriteString("| Secure token storage | `core/datastore/`, `docs/STORAGE.md` |\n")
		}
	case kindDesktop:
		if target.Stack.ID == "desktop/electron" {
			b.WriteString("| UI | `src/renderer/`, `docs/UI.md` |\n")
			b.WriteString("| IPC and trusted process | `src/main/`, `src/preload/`, `CONTEXT.md` |\n")
			b.WriteString("| Local data | `src/main/storage/`, `docs/SYNC.md` |\n")
		} else {
			b.WriteString("| UI and actions | `crates/app/`, `docs/UI.md` |\n")
			b.WriteString("| Local data | `crates/storage/`, `docs/SYNC.md` |\n")
			b.WriteString("| API/sync | `crates/api-client/`, `docs/AUTH.md`, `docs/SYNC.md` |\n")
			b.WriteString("| Platform bootstrap | `crates/desktop/` |\n")
		}
	}
}

func writeTargetRows(b *strings.Builder, target resolve.TargetPlan) {
	switch target.Stack.Kind {
	case kindBackend:
		fmt.Fprintf(b, "| API service | `%s/README.md`, `%s/CONTEXT.md` |\n", target.Path, target.Path)
	case kindFrontend:
		fmt.Fprintf(b, "| Web app and BFF | `%s/README.md`, `%s/CONTEXT.md` |\n", target.Path, target.Path)
	case kindMobile:
		label := "Android app"
		if target.Stack.ID == iosStackID {
			label = "iOS app"
		}

		fmt.Fprintf(b, "| %s | `%s/README.md`, `%s/CONTEXT.md` |\n", label, target.Path, target.Path)
	case kindDesktop:
		fmt.Fprintf(b, "| Desktop app | `%s/README.md`, `%s/CONTEXT.md` |\n", target.Path, target.Path)
	}
}

func targetStackKind(plan *resolve.Plan, kind string) string {
	for _, target := range allPlanTargets(plan) {
		if target.Stack.Kind == kind {
			return target.Stack.ID
		}
	}

	return ""
}

func desktopStack(plan *resolve.Plan) string {
	for _, target := range allPlanTargets(plan) {
		if target.Stack.Kind == "desktop" {
			return target.Stack.ID
		}
	}

	return ""
}

func allPlanTargets(plan *resolve.Plan) []resolve.TargetPlan {
	targets := slices.Clone(plan.Targets)
	if plan.Root != nil {
		targets = append(targets, *plan.Root)
	}

	return targets
}

func targetPaths(plan *resolve.Plan) []string {
	targets := allPlanTargets(plan)

	paths := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.Path != "." {
			paths = append(paths, target.Path)
		}
	}

	return paths
}

func hasFeature(plan *resolve.Plan, featureID string) bool {
	for _, target := range allPlanTargets(plan) {
		for _, overlay := range target.Overlays {
			if overlay.FeatureID == featureID {
				return true
			}
		}
	}

	return false
}
