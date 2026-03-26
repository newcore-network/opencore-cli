package updater

import "testing"

func TestNormalizeChannel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "default", input: "", want: ChannelStable},
		{name: "stable", input: "stable", want: ChannelStable},
		{name: "stable upper", input: " STABLE ", want: ChannelStable},
		{name: "beta", input: "beta", want: ChannelBeta},
		{name: "unknown", input: "nightly", want: ChannelStable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeChannel(tt.input); got != tt.want {
				t.Fatalf("NormalizeChannel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReleaseAllowedForChannel(t *testing.T) {
	stable := Release{TagName: "v1.2.0"}
	beta := Release{TagName: "v1.3.0-beta.1", Prerelease: true}
	rc := Release{TagName: "v1.3.0-rc.1", Prerelease: true}
	draft := Release{TagName: "v1.4.0-beta.1", Prerelease: true, Draft: true}

	if !releaseAllowedForChannel(stable, ChannelStable) {
		t.Fatalf("expected stable release on stable channel")
	}
	if releaseAllowedForChannel(beta, ChannelStable) {
		t.Fatalf("expected beta prerelease to be excluded from stable channel")
	}
	if !releaseAllowedForChannel(stable, ChannelBeta) {
		t.Fatalf("expected stable release on beta channel")
	}
	if !releaseAllowedForChannel(beta, ChannelBeta) {
		t.Fatalf("expected beta prerelease on beta channel")
	}
	if releaseAllowedForChannel(rc, ChannelBeta) {
		t.Fatalf("expected non-beta prerelease to be excluded from beta channel")
	}
	if releaseAllowedForChannel(draft, ChannelBeta) {
		t.Fatalf("expected draft release to be excluded")
	}
}

func TestSelectReleaseForChannel(t *testing.T) {
	releases := []Release{
		{TagName: "v1.2.0"},
		{TagName: "v1.3.0-beta.1", Prerelease: true},
		{TagName: "v1.3.0-beta.2", Prerelease: true},
		{TagName: "v1.3.0-rc.1", Prerelease: true},
	}

	stable, err := selectReleaseForChannel(releases, ChannelStable)
	if err != nil {
		t.Fatalf("stable select error: %v", err)
	}
	if stable.TagName != "v1.2.0" {
		t.Fatalf("stable TagName = %q, want %q", stable.TagName, "v1.2.0")
	}

	beta, err := selectReleaseForChannel(releases, ChannelBeta)
	if err != nil {
		t.Fatalf("beta select error: %v", err)
	}
	if beta.TagName != "v1.3.0-beta.2" {
		t.Fatalf("beta TagName = %q, want %q", beta.TagName, "v1.3.0-beta.2")
	}
}
