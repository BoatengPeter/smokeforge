package cli

import (
	"context"
	"errors"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
)

type mutationProject struct {
	root  string
	state *project.State
	lock  *project.Lock
}

func loadMutationProject(dir string) (*mutationProject, error) {
	root, err := project.FindRoot(dir)
	if err != nil {
		return nil, err
	}

	state, err := project.LoadState(root)
	if err != nil {
		return nil, err
	}

	lock, err := project.LoadLock(root)
	if err != nil {
		return nil, err
	}

	return &mutationProject{root: root, state: state, lock: lock}, nil
}

func loadMutationRegistry(
	ctx context.Context,
	opts *options,
	projectContext *mutationProject,
) (*registry.Result, error) {
	source := projectContext.state.Registry.Source
	if opts.registry != "" {
		source = opts.registry
	}

	if source == "" {
		return nil, errors.New("project does not record a registry source; pass --registry")
	}

	return registry.Load(ctx, opts.registryOptions(source, projectContext.lock.Registry.Tag))
}
