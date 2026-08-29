package commands

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateBranchRejectsOptionAndInvalidRefs(t *testing.T) {
	for _, branch := range []string{"main", "feature/security", "release-1.0"} {
		if err := validateBranch(branch); err != nil {
			t.Errorf("expected branch %q to be valid: %v", branch, err)
		}
	}
	for _, branch := range []string{"--upload-pack=evil", "feature..test", "feature//test", "bad branch", "main.lock"} {
		if err := validateBranch(branch); err == nil {
			t.Errorf("expected branch %q to be invalid", branch)
		}
	}
}

func TestValidateGitHubItemRejectsTraversal(t *testing.T) {
	for _, item := range []GitHubContent{
		{Name: "../escape", Path: "resources/escape", Type: "file"},
		{Name: "safe.txt", Path: "other/safe.txt", Type: "file"},
		{Name: `..\escape`, Path: `resources/chat/..\escape`, Type: "file"},
	} {
		if err := validateGitHubItem("resources/chat", item); err == nil {
			t.Errorf("expected item %#v to be rejected", item)
		}
	}
}

func TestReadLimitedRejectsOversizedResponse(t *testing.T) {
	if _, err := readLimited(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("expected response size error")
	}
}

func TestDownloadFileChecksStatusAndDoesNotCreateFile(t *testing.T) {
	originalClient := cloneHTTPClient
	cloneHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("missing")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})}
	t.Cleanup(func() { cloneHTTPClient = originalClient })

	budget := int64(cloneBodyLimit)
	target := t.TempDir() + "/download.txt"
	err := downloadFile(context.Background(), "https://raw.githubusercontent.com/example/repo/main/file", target, &budget)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestCloneGETUsesRequestContext(t *testing.T) {
	originalClient := cloneHTTPClient
	cloneHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	t.Cleanup(func() { cloneHTTPClient = originalClient })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cloneGET(ctx, "https://api.github.com/test"); err == nil {
		t.Fatal("expected canceled request error")
	}
}
