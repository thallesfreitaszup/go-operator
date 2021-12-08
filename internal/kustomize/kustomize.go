package kustomize

import (
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kustomize/v4/commands/build"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type KustomizeWrapper struct {
	Kustomizer *krusty.Kustomizer
	Filesys    filesys.FileSystem
}

func New() KustomizeWrapper {
	fsys := filesys.MakeFsOnDisk()

	kustomize := krusty.MakeKustomizer(
		build.HonorKustomizeFlags(krusty.MakeDefaultOptions()),
	)
	return KustomizeWrapper{Kustomizer: kustomize, Filesys: fsys}
}
