# smokeforge

Scaffold backend, frontend, mobile, desktop, and fullstack projects from a
versioned template registry with composable feature overlays.

## Status

Foundation: registry manifest loading, validation, caching, and inspection
commands. Project generation lands in the next milestone.

## Build

    make build
    ./bin/smokeforge version
    ./bin/smokeforge list stacks --registry ./internal/manifest/testdata

## Development

    make test
    make lint
