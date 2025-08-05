package gemini

import (
	"context"
	"emperror.dev/errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/je4/kilib/pkg/ki"
	"google.golang.org/genai"
	"io/fs"
	"time"
)

var TokenFields = []string{
	"PromptTokenCount",
	"CandidatesTokenCount",
	"CachedContentTokenCount",
	"TotalTokenCount",
}

var DriverName = "google"

func NewDriver(model, apikey string) (*Driver, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apikey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot create genai client")
	}

	return &Driver{
		client: client,
		model:  model,
	}, nil
}

type Driver struct {
	client *genai.Client
	model  string
	cache  *genai.CachedContent
}

func (d *Driver) CreateCache(ctx context.Context, context []string, ttl time.Duration) (err error) {
	if err := d.ClearCache(ctx); err != nil {
		return errors.Wrap(err, "cannot clear cache")
	}

	cc := &genai.CreateCachedContentConfig{
		TTL: ttl,
		Contents: []*genai.Content{&genai.Content{
			Parts: make([]*genai.Part, len(context)),
		},
		},
	}

	for i, c := range context {
		cc.Contents[0].Parts[i] = &genai.Part{
			Text: c,
		}
	}
	d.cache, err = d.client.Caches.Create(ctx, d.model, cc)
	if err != nil {
		return errors.Wrap(err, "cannot create cache")
	}
	return nil
}

func (d *Driver) ClearCache(ctx context.Context) error {
	if d.cache != nil {
		_, err := d.client.Caches.Delete(ctx, d.cache.Name, nil)
		if err != nil {
			return errors.Wrapf(err, "cannot delete cache %s", d.cache.Name)
		}
	}
	d.cache = nil
	return nil
}

func (d *Driver) QueryWithText(ctx context.Context, input string, context []string) ([]string, map[string]int64, error) {
	//	model := d.client.GenerativeModel(d.model)
	parts := make([]*genai.Part, len(context)+1)
	parts[0] = &genai.Part{Text: input}
	for i, c := range context {
		parts[i+1] = &genai.Part{Text: c}
	}
	resp, err := d.client.Models.GenerateContent(ctx, d.model, []*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate content")
	}
	var result []string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				result = append(result, fmt.Sprintf("%v", part))
			}
		}
	}
	return result, map[string]int64{
		"PromptTokenCount":        int64(resp.UsageMetadata.PromptTokenCount),
		"CandidatesTokenCount":    int64(resp.UsageMetadata.CandidatesTokenCount),
		"CachedContentTokenCount": int64(resp.UsageMetadata.CachedContentTokenCount),
		"TotalTokenCount":         int64(resp.UsageMetadata.TotalTokenCount),
	}, nil

}

func (d *Driver) GetModel() string {
	return d.model
}

func (d *Driver) GetName() string {
	return DriverName
}

func (d *Driver) QueryWithImage(ctx context.Context, input string, fsys fs.FS, path string) ([]string, map[string]int64, error) {
	fp, err := fsys.Open(path)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot open file %s", path)
	}
	defer fp.Close()
	file, err := d.client.Files.Upload(ctx, fp, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot upload file")
	}
	defer d.client.Files.Delete(ctx, file.Name, nil)

	parts := make([]*genai.Part, 2)
	parts[0] = &genai.Part{Text: input}
	parts[1] = &genai.Part{FileData: &genai.FileData{
		DisplayName: file.DisplayName,
		FileURI:     file.URI,
		MIMEType:    file.MIMEType,
	}}
	resp, err := d.client.Models.GenerateContent(ctx, d.model, []*genai.Content{{Parts: parts}}, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate content")
	}
	var result []string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				result = append(result, fmt.Sprintf("%v", part))
			}
		}
	}
	return result, map[string]int64{
		"PromptTokenCount":        int64(resp.UsageMetadata.PromptTokenCount),
		"CandidatesTokenCount":    int64(resp.UsageMetadata.CandidatesTokenCount),
		"CachedContentTokenCount": int64(resp.UsageMetadata.CachedContentTokenCount),
		"TotalTokenCount":         int64(resp.UsageMetadata.TotalTokenCount),
	}, nil
}

var _ ki.Interface = &Driver{}
