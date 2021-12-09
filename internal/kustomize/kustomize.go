package kustomize

import (
	"fmt"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/kustomize/v4/commands/build"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type KustomizeWrapper struct {
	Kustomizer *krusty.Kustomizer
	Filesys    filesys.FileSystem
}

func (w KustomizeWrapper) RenderManifests(chart string) (resmap.ResMap, error) {
	response, err := w.Kustomizer.Run(w.Filesys, chart)
	if err != nil {
		return nil, fmt.Errorf("error running build of kustomize:  %w", err)
	}
	return response, nil
}

func New() KustomizeWrapper {
	fsys := filesys.MakeFsOnDisk()

	kustomize := krusty.MakeKustomizer(
		build.HonorKustomizeFlags(krusty.MakeDefaultOptions()),
	)
	return KustomizeWrapper{Kustomizer: kustomize, Filesys: fsys}
}
