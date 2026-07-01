package attachmentclient

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type MemoryFileClient struct {
	mu   sync.Mutex
	next int
	data map[string][]byte
}

func NewMemoryFileClient() *MemoryFileClient {
	return &MemoryFileClient{data: map[string][]byte{}}
}

func (c *MemoryFileClient) Upload(_ context.Context, name, contentType string, size int64, body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	ref := fmt.Sprintf("qa-file-%d", c.next)
	c.data[ref] = data
	return ref, nil
}

func (c *MemoryFileClient) Read(_ context.Context, fileRef string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.data[fileRef]
	if !ok {
		return nil, fmt.Errorf("file not found")
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func (c *MemoryFileClient) Delete(_ context.Context, fileRef string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, fileRef)
	return nil
}

type PlainTextParserClient struct{}

func (PlainTextParserClient) Parse(_ context.Context, filename, contentType string, data []byte) (service.ParsedAttachment, error) {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return service.ParsedAttachment{}, fmt.Errorf("document is empty")
	}
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	chunks := make([]service.ParsedAttachmentChunk, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		chunks = append(chunks, service.ParsedAttachmentChunk{
			PageNumber: 1,
			Content:    part,
		})
		if i >= 49 {
			break
		}
	}
	if len(chunks) == 0 {
		chunks = []service.ParsedAttachmentChunk{{PageNumber: 1, Content: text}}
	}
	return service.ParsedAttachment{PageCount: 1, Chunks: chunks}, nil
}
