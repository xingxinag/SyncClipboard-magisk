package config

import "testing"

func TestNormalizeSyncModeUsesNewFlags(t *testing.T) {
	cfg := &Config{AutoUploadEnabled: true, AutoDownloadEnabled: false}

	cfg.NormalizeSyncMode()

	if !cfg.Enabled {
		t.Fatalf("expected enabled true when either auto flag true")
	}

	cfg = &Config{AutoUploadEnabled: false, AutoDownloadEnabled: false}
	cfg.NormalizeSyncMode()
	if cfg.Enabled {
		t.Fatalf("expected enabled false when both auto flags false")
	}
}

func TestDefaultConfigDisablesAutoModes(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AutoUploadEnabled || cfg.AutoDownloadEnabled || cfg.Enabled {
		t.Fatalf("expected default config auto modes disabled")
	}
}
