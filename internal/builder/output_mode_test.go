package builder

import "testing"

func TestParseOutputMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    OutputMode
		wantErr bool
	}{
		{name: "default empty", input: "", want: OutputModeAuto},
		{name: "auto", input: "auto", want: OutputModeAuto},
		{name: "tui", input: "tui", want: OutputModeTUI},
		{name: "plain", input: "plain", want: OutputModePlain},
		{name: "trimmed upper", input: "  PLAIN ", want: OutputModePlain},
		{name: "invalid", input: "json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOutputMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
