package kubeconfigfinalizer

import (
	"context"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

func filter(finalizers []string, matching string) []string {
	var ret []string
	for _, f := range finalizers {
		if f != matching {
			ret = append(ret, f)
		}
	}
	return ret
}
