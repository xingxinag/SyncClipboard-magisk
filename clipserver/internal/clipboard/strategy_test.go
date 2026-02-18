package clipboard

import "testing"

func TestDetectClipboardStrategy_SystemHookFirst(t *testing.T) {
	s := detectClipboardStrategy()

	if len(s.readOrder) == 0 || s.readOrder[0].name != "system_hook" {
		t.Fatalf("expected read first method system_hook, got %+v", s.readOrder)
	}
	if len(s.writeOrder) == 0 || s.writeOrder[0].name != "system_hook" {
		t.Fatalf("expected write first method system_hook, got %+v", s.writeOrder)
	}

	for _, m := range s.readOrder {
		if m.name == "apk_helper" || m.name == "shared_file" {
			t.Fatalf("legacy read method should not exist: %s", m.name)
		}
	}
	for _, m := range s.writeOrder {
		if m.name == "apk_helper" || m.name == "shared_file" {
			t.Fatalf("legacy write method should not exist: %s", m.name)
		}
	}
}
