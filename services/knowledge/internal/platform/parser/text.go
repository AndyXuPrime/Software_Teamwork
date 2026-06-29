package parser

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/knowledge/internal/service"
)

const maxParsedTextBytes = 8 << 20

type TextParser struct{}

func NewTextParser() *TextParser {
	return &TextParser{}
}

func (p *TextParser) Parse(ctx context.Context, input service.ParseInput) (service.ParsedDocument, error) {
	if err := ctx.Err(); err != nil {
		return service.ParsedDocument{}, err
	}
	data, err := io.ReadAll(io.LimitReader(input.Body, maxParsedTextBytes+1))
	if err != nil {
		return service.ParsedDocument{}, err
	}
	if len(data) > maxParsedTextBytes {
		return service.ParsedDocument{}, fmt.Errorf("document is too large for parser")
	}
	if !utf8.Valid(data) {
		return service.ParsedDocument{}, fmt.Errorf("document text encoding is not supported")
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return service.ParsedDocument{}, fmt.Errorf("document is empty")
	}
	return service.ParsedDocument{
		Content: content,
		Title:   strings.TrimPrefix(firstNonEmptyLine(content), "# "),
	}, nil
}
