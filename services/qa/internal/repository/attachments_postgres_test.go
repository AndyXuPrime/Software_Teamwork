package repository

import "testing"

func TestAttachmentChunkLimitNormalizationSeparatesSearchAndReportSource(t *testing.T) {
	if got := normalizeAttachmentSearchLimit(0); got != defaultAttachmentSearchLimit {
		t.Fatalf("normalizeAttachmentSearchLimit(0) = %d, want %d", got, defaultAttachmentSearchLimit)
	}
	if got := normalizeAttachmentSearchLimit(200); got != maxAttachmentSearchLimit {
		t.Fatalf("normalizeAttachmentSearchLimit(200) = %d, want %d", got, maxAttachmentSearchLimit)
	}
	if got := normalizeAttachmentReportChunkLimit(0); got != defaultAttachmentReportChunkLimit {
		t.Fatalf("normalizeAttachmentReportChunkLimit(0) = %d, want %d", got, defaultAttachmentReportChunkLimit)
	}
	if got := normalizeAttachmentReportChunkLimit(200); got != 200 {
		t.Fatalf("normalizeAttachmentReportChunkLimit(200) = %d, want 200", got)
	}
	if got := normalizeAttachmentReportChunkLimit(500); got != maxAttachmentReportChunkLimit {
		t.Fatalf("normalizeAttachmentReportChunkLimit(500) = %d, want %d", got, maxAttachmentReportChunkLimit)
	}
}
