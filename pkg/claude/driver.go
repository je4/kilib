package claude

import (
	"context"
	"emperror.dev/errors"
	"github.com/je4/kilib/pkg/ki"
	"github.com/liushuangls/go-anthropic/v2"
	"io/fs"
	"mime"
	"net/http"
)

var TokenFields = []string{
	"InputTokens",
	"OutputTokens",
	"CacheCreationInputTokens",
	"CacheReadInputTokens",
}

var DriverName = "anthropic"

func NewDriver(model, apikey string) (*Driver, error) {
	client := anthropic.NewClient(apikey)
	return &Driver{
		client: client,
		model:  model,
	}, nil
}

type Driver struct {
	client *anthropic.Client
	model  string
}

func (d *Driver) QueryWithText(ctx context.Context, input string, context []string) ([]string, map[string]int64, error) {
	//TODO implement me
	panic("implement me")
}

func (d *Driver) GetModel() string {
	return d.model
}

func (d *Driver) GetName() string {
	return DriverName
}

func (d *Driver) QueryWithImage(ctx context.Context, input string, fsys fs.FS, path string) ([]string, map[string]int64, error) {
	imgData, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot read file %s", path)
	}
	contentType := http.DetectContentType(imgData)
	contentType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "cannot parse content type %s", contentType)
	}
	/*
		buf := bytes.NewBuffer(nil)
		enc := base64.NewEncoder(base64.StdEncoding, buf)
		if _, err := enc.Write(imgData); err != nil {
			return "", nil, errors.Wrap(err, "cannot encode file")
		}
	*/
	resp, err := d.client.CreateMessages(context.Background(), anthropic.MessagesRequest{
		Model: anthropic.Model(d.model),
		Messages: []anthropic.Message{
			{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewImageMessageContent(anthropic.MessageContentSource{
						Type:      "base64",
						MediaType: contentType,
						Data:      imgData,
					}),
					anthropic.NewTextMessageContent(input),
				},
			},
		},
		MaxTokens: 5000,
	})
	if err != nil {
		var e *anthropic.APIError
		if errors.As(err, &e) {
			return nil, nil, errors.Errorf("Messages error, type: %s, message: %s", e.Type, e.Message)
		} else {
			return nil, nil, errors.Wrap(err, "cannot generate content")
		}
	}
	if len(resp.Content) == 0 {
		return nil, nil, errors.New("no completion")
	}
	comp := make([]string, 0)
	for _, part := range resp.Content {
		if part.Text != nil {
			comp = append(comp, *part.Text)
		}
	}
	if resp.StopReason != "" {
		comp = append(comp, "stop: "+string(resp.StopReason))
	}
	return comp, map[string]int64{
		"InputTokens":              int64(resp.Usage.InputTokens),
		"OutputTokens":             int64(resp.Usage.OutputTokens),
		"CacheCreationInputTokens": int64(resp.Usage.CacheCreationInputTokens),
		"CacheReadInputTokens":     int64(resp.Usage.CacheReadInputTokens),
	}, nil
}

var _ ki.Interface = &Driver{}
