package txadmin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionCookieFromSetCookie(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    string
		ok      bool
	}{
		{
			name:    "current txAdmin session cookie",
			headers: []string{"txa:sess:abcdef123456=11111111-1111-1111-1111-111111111111; Path=/; HttpOnly"},
			want:    "txa:sess:abcdef123456",
			ok:      true,
		},
		{
			name: "theme cookie is ignored in favor of session",
			headers: []string{
				"txa:theme=dark; Path=/",
				"txa:sess:abcdef123456=11111111-1111-1111-1111-111111111111; Path=/; HttpOnly",
			},
			want: "txa:sess:abcdef123456",
			ok:   true,
		},
		{
			name:    "legacy profile cookie",
			headers: []string{"tx:default=legacy-session; Path=/"},
			want:    "tx:default",
			ok:      true,
		},
		{
			name:    "theme cookie alone is not a session",
			headers: []string{"txa:theme=dark; Path=/"},
			ok:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, _, ok := sessionCookieFromSetCookie(tt.headers)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if gotName != tt.want {
				t.Fatalf("cookie name = %q, want %q", gotName, tt.want)
			}
		})
	}
}

func TestLoginAcceptsCurrentTxAdminSessionCookie(t *testing.T) {
	const csrf = "csrf-token"
	const sessID = "11111111-1111-1111-1111-111111111111"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/password" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Set-Cookie", "txa:theme=dark; Path=/")
		w.Header().Add("Set-Cookie", "txa:sess:abcdef123456="+sessID+"; Path=/; HttpOnly")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"csrfToken":"` + csrf + `"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := NewClient(srv.URL, "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Login(); err != nil {
		t.Fatal(err)
	}

	if client.csrfToken != csrf {
		t.Fatalf("csrf = %q, want %q", client.csrfToken, csrf)
	}
	if client.session == nil || client.session.CookieName != "txa:sess:abcdef123456" || client.session.Cookie != sessID {
		t.Fatalf("session = %#v", client.session)
	}
	if client.sessionCookie != "txa:sess:abcdef123456="+sessID {
		t.Fatalf("session cookie = %q", client.sessionCookie)
	}
}
