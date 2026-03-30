package artifact

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

const papermcBaseURL = "https://fill.papermc.io/v3"

func init() {
	registerJarResolver(resolvePaperMCJar)
}

func resolvePaperMCJar(src *config.ServerSource, _ *http.Client) (source, bool) {
	if src.PaperMC == nil {
		return nil, false
	}
	return &papermcSource{
		project: src.PaperMC.Project,
		version: src.PaperMC.Version,
		build:   src.PaperMC.Build,
	}, true
}

type papermcSource struct {
	project string
	version string
	build   int
}

type papermcBuildResponse struct {
	Downloads map[string]papermcDownload `json:"downloads"`
}

type papermcDownload struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *papermcSource) download(client *http.Client, destPath string, onProgress ProgressFunc) error {
	downloadURL, err := s.resolveDownloadURL(client)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	return writeHTTPResponse(req, client, destPath, onProgress)
}

func (s *papermcSource) resolveDownloadURL(client *http.Client) (string, error) {
	apiURL := fmt.Sprintf("%s/projects/%s/versions/%s/builds/%d", papermcBaseURL, s.project, s.version, s.build)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "mcserver-image-builder (go.junhyung.kr/mcserver-image-builder)")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching PaperMC API for %s %s build %d: %w", s.project, s.version, s.build, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PaperMC API returned status %d for %s %s build %d", resp.StatusCode, s.project, s.version, s.build)
	}

	var buildResp papermcBuildResponse
	if err := json.NewDecoder(resp.Body).Decode(&buildResp); err != nil {
		return "", fmt.Errorf("decoding PaperMC API response: %w", err)
	}

	dl, ok := buildResp.Downloads["server:default"]
	if !ok {
		dl, ok = buildResp.Downloads["proxy:default"]
		if !ok {
			return "", fmt.Errorf("no download URL found in PaperMC API response for %s %s build %d", s.project, s.version, s.build)
		}
	}

	return dl.URL, nil
}

func (s *papermcSource) cacheKey() string {
	return cacheKey("papermc", s.project, s.version, fmt.Sprintf("%d", s.build))
}

func (s *papermcSource) describe() string {
	return fmt.Sprintf("papermc:%s/%s/%d", s.project, s.version, s.build)
}
