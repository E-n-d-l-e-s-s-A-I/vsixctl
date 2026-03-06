package cliutils

import (
	"strings"
	"testing"
)

func TestPacmanProgressBarDraw(t *testing.T) {
	// fixedOverhead=87, barWidth=97-87=10
	const termWidth = 97

	paddedExt := "ext-a" + strings.Repeat(" ", 30)   // 35 символов
	paddedGit := "gitlens" + strings.Repeat(" ", 28) // 35 символов

	tests := []struct {
		name          string
		terminalWidth int
		label         string
		downloaded    int64
		total         int64
		want          string
	}{
		{
			name:          "empty",
			terminalWidth: termWidth,
			label:         "ext-a",
			downloaded:    0,
			total:         1024 * 1024,
			want:          paddedExt + "  [          ]  0%  0.0 MiB / 1.0 MiB",
		},
		{
			name:          "half",
			terminalWidth: termWidth,
			label:         "ext-a",
			downloaded:    512 * 1024,
			total:         1024 * 1024,
			want:          paddedExt + "  [#####     ]  50%  0.5 MiB / 1.0 MiB",
		},
		{
			name:          "full",
			terminalWidth: termWidth,
			label:         "ext-a",
			downloaded:    1024 * 1024,
			total:         1024 * 1024,
			want:          paddedExt + "  [##########]  100%  1.0 MiB / 1.0 MiB",
		},
		{
			name:          "unknown_total",
			terminalWidth: termWidth,
			label:         "ext-a",
			downloaded:    512 * 1024,
			total:         0,
			want:          paddedExt + "  0.5 MiB",
		},
		{
			name:          "negative_total",
			terminalWidth: termWidth,
			label:         "ext-a",
			downloaded:    100,
			total:         -1,
			want:          paddedExt + "  0.0 MiB",
		},
		{
			name:          "different_label",
			terminalWidth: termWidth,
			label:         "gitlens",
			downloaded:    256 * 1024,
			total:         1024 * 1024,
			want:          paddedGit + "  [##        ]  25%  0.2 MiB / 1.0 MiB",
		},
		{
			name:          "narrow_terminal_no_bar",
			terminalWidth: 50,
			label:         "ext-a",
			downloaded:    512 * 1024,
			total:         1024 * 1024,
			want:          paddedExt + "  50%  0.5 MiB / 1.0 MiB",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			pb := NewPacmanProgressBar()
			got := pb.Draw(testCase.label, testCase.downloaded, testCase.total, testCase.terminalWidth)

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

// Проверяет что фабрика создаёт стиль по имени и возвращает ошибку для неизвестных стилей
func TestNewProgressBarStyle(t *testing.T) {
	tests := []struct {
		name    string
		style   string
		wantErr bool
	}{
		{
			name:    "pacman",
			style:   "pacman",
			wantErr: false,
		},
		{
			name:    "unknown_style",
			style:   "unknown",
			wantErr: true,
		},
		{
			name:    "empty_style",
			style:   "",
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			style, err := NewProgressBarStyle(testCase.style)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if style == nil {
				t.Fatal("expected non-nil style")
			}
		})
	}
}
