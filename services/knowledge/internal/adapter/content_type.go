package adapter

import (
	"mime"
	"path/filepath"
	"strings"
)

const genericDocumentContentType = "application/octet-stream"

var documentContentTypesByExtension = map[string]string{
	".pdf":      "application/pdf",
	".txt":      "text/plain",
	".md":       "text/markdown",
	".markdown": "text/markdown",
	".mdx":      "text/markdown",
	".doc":      "application/msword",
	".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ppt":      "application/vnd.ms-powerpoint",
	".pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".xls":      "application/vnd.ms-excel",
	".xlsx":     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".csv":      "text/csv",
	".png":      "image/png",
	".jpg":      "image/jpeg",
	".jpeg":     "image/jpeg",
	".gif":      "image/gif",
	".bmp":      "image/bmp",
	".webp":     "image/webp",
}

func normalizedDocumentContentType(raw map[string]interface{}) *string {
	genericContentType := ""
	for _, key := range []string{"content_type", "contentType", "mime_type", "mimeType"} {
		if contentType := normalizeDocumentMediaType(stringField(raw, key)); contentType != "" {
			if contentType == genericDocumentContentType {
				genericContentType = contentType
				continue
			}
			return stringPtr(contentType)
		}
	}

	for _, key := range []string{"name", "filename", "file_name"} {
		if contentType := documentContentTypeFromFilename(stringField(raw, key)); contentType != "" {
			return stringPtr(contentType)
		}
	}

	if contentType := documentContentTypeFromExtension(stringField(raw, "suffix")); contentType != "" {
		return stringPtr(contentType)
	}

	if genericContentType != "" {
		return stringPtr(genericContentType)
	}

	if contentType := documentContentTypeFromRuntimeType(stringField(raw, "type")); contentType != "" {
		return stringPtr(contentType)
	}

	return nil
}

func stringPtr(value string) *string {
	return &value
}

func documentContentTypeFromFilename(filename string) string {
	ext := filepath.Ext(strings.TrimSpace(filename))
	return documentContentTypeFromExtension(ext)
}

func documentContentTypeFromExtension(extension string) string {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension == "" {
		return ""
	}
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	if contentType := documentContentTypesByExtension[extension]; contentType != "" {
		return contentType
	}
	return ""
}

func documentContentTypeFromRuntimeType(runtimeType string) string {
	runtimeType = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(runtimeType)), ".")
	if runtimeType == "" {
		return ""
	}
	if contentType := normalizeDocumentMediaType(runtimeType); contentType != "" {
		return contentType
	}
	switch runtimeType {
	case "pdf", "txt", "md", "markdown", "mdx", "docx", "ppt", "pptx", "xls", "xlsx", "csv",
		"png", "jpg", "jpeg", "gif", "bmp", "webp":
		return documentContentTypeFromExtension(runtimeType)
	default:
		return ""
	}
}

func normalizeDocumentMediaType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		mediaType = strings.SplitN(value, ";", 2)[0]
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if !strings.Contains(mediaType, "/") {
		return ""
	}
	return mediaType
}
