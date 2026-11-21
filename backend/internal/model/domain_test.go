package model

import "testing"

func TestIsValidProjectBackgroundColor(t *testing.T) {
	for _, c := range append([]string{""}, ProjectBackgroundColors...) {
		if !IsValidProjectBackgroundColor(c) {
			t.Errorf("expected %q to be valid", c)
		}
	}
	for _, c := range []string{"sky", "RED", "#ffffff", "DEFAULT "} {
		if IsValidProjectBackgroundColor(c) {
			t.Errorf("expected %q to be invalid", c)
		}
	}
	if DefaultProjectSettings().BackgroundColor != "DEFAULT" {
		t.Errorf("default background color should be DEFAULT")
	}
}
