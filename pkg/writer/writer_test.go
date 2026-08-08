package writer

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	headers := []string{"CONTEXT", "NAME", "UP-TO-DATE", "ACCESS MODES"}
	matrix := [][]string{
		{"staging", "app-1", "1", "RWO"},
		{"production", "app-2", "2", "RWX"},
	}

	buf := bytes.NewBuffer(nil)
	if err := WriteJSON(buf, headers, matrix); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	want := `[
  {
    "CONTEXT": "staging",
    "NAME": "app-1",
    "UP-TO-DATE": "1",
    "ACCESS MODES": "RWO"
  },
  {
    "CONTEXT": "production",
    "NAME": "app-2",
    "UP-TO-DATE": "2",
    "ACCESS MODES": "RWX"
  }
]
`
	if got := buf.String(); got != want {
		t.Errorf("WriteJSON() = %q, want %q", got, want)
	}

	var decoded []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(decoded) != 2 {
		t.Errorf("decoded %d objects, want 2", len(decoded))
	}
}

func TestWriteJSONEmpty(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	if err := WriteJSON(buf, []string{"CONTEXT", "NAME"}, nil); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	if got := buf.String(); got != "[]\n" {
		t.Errorf("WriteJSON() = %q, want %q", got, "[]\n")
	}
}
