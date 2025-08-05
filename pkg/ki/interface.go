package ki

import (
	"context"
	"io/fs"
	"time"
)

type Interface interface {
	QueryWithImage(ctx context.Context, input string, fsys fs.FS, path string) ([]string, map[string]int64, error)
	QueryWithText(ctx context.Context, input string, context []string) ([]string, map[string]int64, error)
	CreateCache(ctx context.Context, context []string, ttl time.Duration) error
	ClearCache(ctx context.Context) error
	GetModel() string
	GetName() string
}
