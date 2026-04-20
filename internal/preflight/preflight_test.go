package preflight

import "testing"

func TestCheckRejectsUnsupportedBrowser(t *testing.T) {
	checker := Checker{
		DetectDefaultBrowser: func() (string, error) { return "safari", nil },
		CheckPort8080:        func() error { return nil },
		CheckWritableRoot:    func() error { return nil },
	}

	result, err := checker.Run()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Supported {
		t.Fatal("expected unsupported browser to fail support check")
	}
	if result.Reason == "" {
		t.Fatal("expected actionable failure reason")
	}
}
