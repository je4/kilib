package ki

import (
	"context"
	"io/fs"
)

type Interface interface {
	QueryWithImage(ctx context.Context, input string, fsys fs.FS, path string) ([]string, map[string]int64, error)
	QueryWithText(ctx context.Context, input string, context []string) ([]string, map[string]int64, error)
	GetModel() string
	GetName() string
}
