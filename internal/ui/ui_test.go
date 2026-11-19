package ui

import "testing"

func TestNonInteractive(t *testing.T) {
	NonInteractive = true
	defer func() { NonInteractive = false }()
	if AskYesNo("Run this plan?", true) {
		t.Error("a non-interactive question must be answered no, even when the default is yes")
	}
	if Choose(3) != 0 {
		t.Error("a non-interactive choice must be skip")
	}
}
