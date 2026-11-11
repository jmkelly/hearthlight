package theme

import (
	"strings"
	"testing"
)

func TestParseColor(t *testing.T) {
	valid := []string{
		"#abc",
		"#abcd",
		"#a1b2c3",
		"#a1b2c3d4",
		"rgb(1, 2, 3)",
		"rgba(1, 2, 3, 0.5)",
		"rgba(255, 255, 255, 1)",
	}
	for _, value := range valid {
		if !validColor(value) {
			t.Errorf("validColor(%q) = false, want true", value)
		}
	}
	invalid := []string{
		"",
		"#",
		"#12",
		"#12345",
		"#1234567",
		"#gggggg",
		"red",
		"rgb(256, 0, 0)",
		"rgba(0, 0, 0, 1.5)",
		"rgba(0, 0, 0, -0.1)",
		"rgba(0, 0, 0, nope)",
		"color(1, 2, 3)",
	}
	for _, value := range invalid {
		if validColor(value) {
			t.Errorf("validColor(%q) = true, want false", value)
		}
	}
}

func TestValidationErrorFormatting(t *testing.T) {
	withPath := (&ValidationError{Category: ErrInvalidColor, Path: "variant.palette.bg", Detail: "bad"}).Error()
	if !strings.Contains(withPath, "invalid_color") || !strings.Contains(withPath, "variant.palette.bg") || !strings.Contains(withPath, "bad") {
		t.Fatalf("error = %q, want category, path, and detail", withPath)
	}
	withoutPath := (&ValidationError{Category: ErrTooLarge, Detail: "too big"}).Error()
	if !strings.Contains(withoutPath, "too_large") || !strings.Contains(withoutPath, "too big") {
		t.Fatalf("error = %q, want category and detail", withoutPath)
	}
}
