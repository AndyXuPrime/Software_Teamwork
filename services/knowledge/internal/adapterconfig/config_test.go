package adapterconfig

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("KNOWLEDGE_HTTP_ADDR", "")
	t.Setenv("VENDOR_RUNTIME_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Fatalf("HTTPAddr=%q", cfg.HTTPAddr)
	}
	if cfg.VendorRuntimeURL != DefaultVendorRuntimeURL {
		t.Fatalf("VendorRuntimeURL=%q", cfg.VendorRuntimeURL)
	}
}

func TestLoadCustomVendorURL(t *testing.T) {
	t.Setenv("VENDOR_RUNTIME_URL", "http://knowledge-vendor:9380/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.VendorRuntimeURL != "http://knowledge-vendor:9380" {
		t.Fatalf("VendorRuntimeURL=%q", cfg.VendorRuntimeURL)
	}
}
