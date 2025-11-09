package format

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefaultTextFormatter_WrapLines(t *testing.T) {
	f := &DefaultTextFormatter{}
	tests := []struct {
		name     string
		text     string
		width    uint
		expected []string
	}{
		{name: "empty", text: "", width: 120, expected: []string{""}},
		{
			name:     "short line within width",
			text:     "short line",
			width:    120,
			expected: []string{"short line"},
		},
		{
			name:     "text exactly at width",
			text:     strings.Repeat("a", 10),
			width:    10,
			expected: []string{strings.Repeat("a", 10)},
		},
		{
			name:     "long word exceeding limit does not wrap",
			text:     "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			width:    5,
			expected: []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		},
		{
			name:     "near the limit",
			text:     "1 22 333 4444",
			width:    3,
			expected: []string{"1", "22", "333", "4444"},
		},
		{
			name:  "long text",
			text:  "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vestibulum vulputate, sapien non hendrerit commodo, nisi orci dictum justo, non iaculis turpis lacus a est.",
			width: 60,
			expected: []string{
				"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
				"Vestibulum vulputate, sapien non hendrerit commodo, nisi",
				"orci dictum justo, non iaculis turpis lacus a est.",
			},
		},
		{
			name:  "multiple paragraphs",
			text:  "para one line 1\n\npara two line 1\npara two line 2",
			width: 80,
			expected: []string{
				"para one line 1",
				"",
				"para two line 1",
				"para two line 2",
			},
		},
		{
			name:  "whitespace-only paragraph",
			text:  "First\n   \nLast",
			width: 80,
			expected: []string{
				"First",
				"   ",
				"Last",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := f.WrapLines(tc.text, tc.width)
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("WrapLines() mismatch (-want +got): \n%s", diff)
			}
		})
	}
}

func TestDefaultTextFormatter_FormatDotenvKeyValue_Success(t *testing.T) {
	f := &DefaultTextFormatter{}
	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{key: "KEY", value: "VALUE", expected: `KEY="VALUE"`},
		{key: "GREETING", value: `He said "hello"`, expected: `GREETING="He said \"hello\""`},
		{key: "EMPTY", value: "", expected: `EMPTY=""`},
		{key: "  TRIMMED_KEY  ", value: "v", expected: `TRIMMED_KEY="v"`},
		{key: "PATH", value: `C:\Program Files\App\"bin"`, expected: `PATH="C:\Program Files\App\\"bin\""`},
		{key: "MULTI", value: "line1\nline2", expected: "MULTI=\"line1\nline2\""},
		{key: "NON_TRIMMED_VALUE", value: "   VALUE   ", expected: `NON_TRIMMED_VALUE="   VALUE   "`},
	}

	for _, tc := range tests {
		t.Run(tc.key+":"+tc.value, func(t *testing.T) {
			got, err := f.FormatDotenvKeyValue(tc.key, tc.value)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

// TODO test error: empty key, space-only key
