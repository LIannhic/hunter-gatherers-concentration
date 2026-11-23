package textutil

import (
	"testing"
)

func TestMeasureWidth(t *testing.T) {
	str1 := "Hello"
	str2 := "Héllô" // Accentuated characters: 5 runes, 7 UTF-8 bytes

	w1 := MeasureWidth(str1)
	w2 := MeasureWidth(str2)

	if w1 != w2 {
		t.Errorf("Expected same width for 5-rune strings, got w1=%d, w2=%d", w1, w2)
	}
}

func TestWrapText(t *testing.T) {
	text := "This is a long sentence that should be wrapped into multiple lines for UI layout."
	lines := WrapText(text, 100)

	if len(lines) <= 1 {
		t.Errorf("Expected text to be wrapped into multiple lines, got %d lines", len(lines))
	}
}
