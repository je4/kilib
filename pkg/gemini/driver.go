package gemini

import (
	"context"
	"emperror.dev/errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/je4/kilib/pkg/ki"
	"google.golang.org/api/option"
	"io/fs"
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
	client, err := genai.NewClient(ctx, option.WithAPIKey(apikey))
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
}

func (d *Driver) QueryWithText(ctx context.Context, input string, context []string) ([]string, map[string]int64, error) {
	model := d.client.GenerativeModel(d.model)
	parts := make([]genai.Part, len(context)+1)
	parts[0] = genai.Text(input)
	for i, c := range context {
		parts[i+1] = genai.Text(c)
	}
	resp, err := model.GenerateContent(ctx, parts...)
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
	file, err := d.client.UploadFile(ctx, uuid.New().String(), fp, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot upload file")
	}
	defer d.client.DeleteFile(ctx, file.Name)

	model := d.client.GenerativeModel(d.model)
	resp, err := model.GenerateContent(ctx,
		genai.FileData{URI: file.URI},
		genai.Text(input))
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
