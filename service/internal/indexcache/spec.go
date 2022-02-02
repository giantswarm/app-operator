package indexcache

import "context"

type Interface interface {
	GetIndex(ctx context.Context, url string) (*Index, error)
}
