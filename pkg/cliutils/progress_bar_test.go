package cliutils

import "testing"

func TestPacmanProgressBarDraw(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		label      string
		downloaded int64
		total      int64
		want       string
	}{
		{
			name:       "empty",
			width:      10,
			label:      "ext-a",
			downloaded: 0,
			total:      1024 * 1024,
			want:       "ext-a  [          ]  0%  0.0 MiB / 1.0 MiB",
		},
		{
			name:       "half",
			width:      10,
			label:      "ext-a",
			downloaded: 512 * 1024,
			total:      1024 * 1024,
			want:       "ext-a  [#####     ]  50%  0.5 MiB / 1.0 MiB",
		},
		{
			name:       "full",
			width:      10,
			label:      "ext-a",
			downloaded: 1024 * 1024,
			total:      1024 * 1024,
			want:       "ext-a  [##########]  100%  1.0 MiB / 1.0 MiB",
		},
		{
			name:       "unknown_total",
			width:      10,
			label:      "ext-a",
			downloaded: 512 * 1024,
			total:      0,
			want:       "ext-a  0.5 MiB",
		},
		{
			name:       "negative_total",
			width:      10,
			label:      "ext-a",
			downloaded: 100,
			total:      -1,
			want:       "ext-a  0.0 MiB",
		},
		{
			name:       "different_bucket_count",
			width:      20,
			label:      "gitlens",
			downloaded: 256 * 1024,
			total:      1024 * 1024,
			want:       "gitlens  [#####               ]  25%  0.2 MiB / 1.0 MiB",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			pb := NewPacmanProgressBar(testCase.width)
			got := pb.Draw(testCase.label, testCase.downloaded, testCase.total)

			if got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
