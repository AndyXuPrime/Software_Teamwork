package adapter

import "testing"

func TestNormalizedDocumentContentTypeUsesCommonFileTypes(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]interface{}
		want string
	}{
		{
			name: "docx filename wins over runtime doc class",
			raw:  map[string]interface{}{"name": "manual.docx", "type": "doc"},
			want: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		},
		{
			name: "pdf filename maps to mime",
			raw:  map[string]interface{}{"name": "policy.pdf", "type": "pdf"},
			want: "application/pdf",
		},
		{
			name: "csv filename maps to mime",
			raw:  map[string]interface{}{"name": "records.csv", "type": "doc"},
			want: "text/csv",
		},
		{
			name: "pptx filename maps to mime",
			raw:  map[string]interface{}{"name": "slides.pptx", "type": "doc"},
			want: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
		{
			name: "image filename maps to mime",
			raw:  map[string]interface{}{"name": "photo.png", "type": "visual"},
			want: "image/png",
		},
		{
			name: "suffix maps to mime",
			raw:  map[string]interface{}{"suffix": "xlsx", "type": "doc"},
			want: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		},
		{
			name: "generic mime defers to filename",
			raw: map[string]interface{}{
				"content_type": "application/octet-stream",
				"name":         "notes.md",
				"type":         "doc",
			},
			want: "text/markdown",
		},
		{
			name: "real mime field is normalized",
			raw:  map[string]interface{}{"content_type": "Application/PDF; charset=binary", "name": "unknown.bin"},
			want: "application/pdf",
		},
		{
			name: "runtime extension-like type is a conservative fallback",
			raw:  map[string]interface{}{"type": "pptx"},
			want: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizedDocumentContentType(tc.raw)
			if got == nil {
				t.Fatalf("content type = nil, want %q", tc.want)
			}
			if *got != tc.want {
				t.Fatalf("content type = %q, want %q", *got, tc.want)
			}
		})
	}
}

func TestNormalizedDocumentContentTypeDoesNotExposeRuntimeDocClass(t *testing.T) {
	got := normalizedDocumentContentType(map[string]interface{}{"type": "doc"})
	if got != nil {
		t.Fatalf("content type = %q, want nil", *got)
	}
}
