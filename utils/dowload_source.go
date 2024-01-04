package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/hashicorp/go-retryablehttp"
)

func DownloadAsBytes(artifact sourcev1.Artifact, httpClient *retryablehttp.Client) (*bytes.Buffer, error) {
	artifactURL := artifact.URL
	if hostname := os.Getenv("SOURCE_CONTROLLER_LOCALHOST"); hostname != "" {
		u, err := url.Parse(artifactURL)
		if err != nil {
			return nil, err
		}
		u.Host = hostname
		artifactURL = u.String()
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifact, error: %w", err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download artifact from %s, status: %s", artifactURL, resp.Status)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if artifact.Size != nil && resp.ContentLength != *artifact.Size {
		return nil, fmt.Errorf("expected artifact size %d, got %d", *artifact.Size, len(buf))
	}

	return bytes.NewBuffer(buf), nil
}
