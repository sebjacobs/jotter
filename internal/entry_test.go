package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntryMarshal_BasicFields(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "checkpoint",
		Content:   "Did some work",
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["timestamp"] != "2026-04-11T10:30:00" {
		t.Errorf("timestamp = %v", m["timestamp"])
	}
	if m["type"] != "checkpoint" {
		t.Errorf("type = %v", m["type"])
	}
	if m["content"] != "Did some work" {
		t.Errorf("content = %v", m["content"])
	}
	if _, ok := m["next"]; ok {
		t.Error("next should be omitted when empty")
	}
}

func TestEntryMarshal_WithNext(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "finish",
		Content:   "Wrapped up",
		Next:      "Continue tomorrow",
	}
	data, _ := json.Marshal(e)
	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["next"] != "Continue tomorrow" {
		t.Errorf("next = %v", m["next"])
	}
}

func TestReadEntries_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.jsonl")
	os.WriteFile(path, []byte(""), 0o644)

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadEntries_NonexistentFile(t *testing.T) {
	entries, err := ReadEntries(filepath.Join(t.TempDir(), "nope.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadEntries_MultipleEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	lines := []string{
		`{"timestamp":"2026-04-11T10:00:00","type":"start","content":"Begin"}`,
		`{"timestamp":"2026-04-11T11:00:00","type":"checkpoint","content":"Progress"}`,
		`{"timestamp":"2026-04-11T12:00:00","type":"finish","content":"Done","next":"Rest"}`,
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Content != "Begin" {
		t.Errorf("entries[0].Content = %q", entries[0].Content)
	}
	if entries[2].Next != "Rest" {
		t.Errorf("entries[2].Next = %q", entries[2].Next)
	}
}

func TestFormatEntry_Basic(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "start",
		Content:   "Hello world",
	}
	out := FormatEntry(e)
	if !strings.Contains(out, "## 2026-04-11 10:30 | start") {
		t.Errorf("heading not found in:\n%s", out)
	}
	if !strings.Contains(out, "Hello world") {
		t.Errorf("content not found in:\n%s", out)
	}
	if strings.Contains(out, "**Next:**") {
		t.Error("should not contain Next when empty")
	}
}

func TestFormatEntry_WithNext(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "finish",
		Content:   "Done",
		Next:      "Pick up testing",
	}
	out := FormatEntry(e)
	if !strings.Contains(out, "**Next:** Pick up testing") {
		t.Errorf("next line not found in:\n%s", out)
	}
}

func TestMarshalJSONL_MatchesPythonFormat(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "start",
		Content:   "Hello world",
	}
	data, err := MarshalJSONL(e)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"timestamp": "2026-04-11T10:30:00", "type": "start", "content": "Hello world"}`
	if string(data) != want {
		t.Errorf("got:  %s\nwant: %s", data, want)
	}
}

func TestMarshalJSONL_WithNext(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "finish",
		Content:   "Done",
		Next:      "Continue tomorrow",
	}
	data, err := MarshalJSONL(e)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"timestamp": "2026-04-11T10:30:00", "type": "finish", "content": "Done", "next": "Continue tomorrow"}`
	if string(data) != want {
		t.Errorf("got:  %s\nwant: %s", data, want)
	}
}

func TestMarshalJSONL_OmitsNextWhenEmpty(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "start",
		Content:   "Hello",
	}
	data, _ := MarshalJSONL(e)
	if strings.Contains(string(data), "next") {
		t.Errorf("should not contain next: %s", data)
	}
}

func TestMarshalJSONL_EscapesNewlines(t *testing.T) {
	e := Entry{
		Timestamp: "2026-04-11T10:30:00",
		Type:      "checkpoint",
		Content:   "Line one\nLine two",
	}
	data, _ := MarshalJSONL(e)
	// Should be a single line with escaped newline
	if strings.Contains(string(data), "\n") {
		t.Errorf("should not contain literal newline: %s", data)
	}
	if !strings.Contains(string(data), `\n`) {
		t.Errorf("should contain escaped newline: %s", data)
	}
}

func TestValidEntryTypes(t *testing.T) {
	for _, typ := range []string{"start", "checkpoint", "note", "break", "finish"} {
		if !IsValidEntryType(typ) {
			t.Errorf("%q should be valid", typ)
		}
	}
	if IsValidEntryType("invalid") {
		t.Error("'invalid' should not be valid")
	}
}
