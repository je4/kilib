package openai

import (
	"bytes"
	"context"
	"emperror.dev/errors"
	"encoding/base64"
	"fmt"
	"github.com/je4/kilib/pkg/ki"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"io/fs"
	"mime"
	"net/http"
)

var TokenFields = []string{
	"CompletionTokens",
	"PromptTokens",
	"TotalTokens",
}

var DriverName = "openai"

func NewDriver(model, apikey string) (*Driver, error) {
	client := openai.NewClient(
		option.WithAPIKey(apikey), // defaults to os.LookupEnv("OPENAI_API_KEY")
	)
	return &Driver{
		client: client,
		model:  model,
	}, nil
}

type Driver struct {
	client *openai.Client
	model  string
}

func (d *Driver) QueryWithText(ctx context.Context, input string, context []string) ([]string, map[string]int64, error) {

	messageParts := make([]openai.ChatCompletionContentPartUnionParam, len(context)+1)
	messageParts[0] = openai.TextPart(input)
	for i, c := range context {
		messageParts[i+1] = openai.TextPart(c)
	}
	param := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessageParts(messageParts...),
		}),
		Model: openai.String(d.model),
	}
	completion, err := d.client.Chat.Completions.New(ctx, param)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate content")
	}
	if len(completion.Choices) == 0 {
		return nil, nil, errors.New("no completion")
	}
	var comp = make([]string, 0)
	for _, cand := range completion.Choices {
		if cand.Message.Content != "" {
			comp = append(comp, cand.Message.Content)
		}
		if cand.Message.Refusal != "" {
			comp = append(comp, cand.Message.Refusal)
		}
	}
	return comp, map[string]int64{
		"CompletionTokens": completion.Usage.CompletionTokens,
		"PromptTokens":     completion.Usage.PromptTokens,
		"TotalTokens":      completion.Usage.TotalTokens,
	}, nil

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
	buf := bytes.NewBuffer(nil)
	enc := base64.NewEncoder(base64.StdEncoding, buf)
	if _, err := enc.Write(imgData); err != nil {
		return nil, nil, errors.Wrap(err, "cannot encode file")
	}

	param := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessageParts(
				openai.TextPart(input),
				openai.ImagePart(fmt.Sprintf("data:%s;base64,%s", contentType, buf.String())),
			),
		}),
		Model: openai.String(d.model),
	}
	completion, err := d.client.Chat.Completions.New(ctx, param)
	if err != nil {
		return nil, nil, errors.Wrap(err, "cannot generate content")
	}
	if len(completion.Choices) == 0 {
		return nil, nil, errors.New("no completion")
	}
	var comp = make([]string, 0)
	for _, cand := range completion.Choices {
		if cand.Message.Content != "" {
			comp = append(comp, cand.Message.Content)
		}
		if cand.Message.Refusal != "" {
			comp = append(comp, cand.Message.Refusal)
		}
	}
	return comp, map[string]int64{
		"CompletionTokens": completion.Usage.CompletionTokens,
		"PromptTokens":     completion.Usage.PromptTokens,
		"TotalTokens":      completion.Usage.TotalTokens,
	}, nil
}

var _ ki.Interface = &Driver{}
