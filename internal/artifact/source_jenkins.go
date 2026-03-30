package artifact

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

func init() {
	registerJarResolver(resolveJenkinsJar)
	registerDownloadResolver(resolveJenkinsDownload)
}

func resolveJenkinsDownload(src *config.DownloadSource, client *http.Client) (source, bool) {
	if src.Jenkins == nil {
		return nil, false
	}
	return newJenkinsSource(src.Jenkins, client), true
}

func resolveJenkinsJar(src *config.ServerSource, client *http.Client) (source, bool) {
	return resolveJenkinsDownload(&src.DownloadSource, client)
}

func newJenkinsSource(cfg *config.JenkinsSource, client *http.Client) *jenkinsSource {
	return &jenkinsSource{
		url:             cfg.URL,
		job:             cfg.Job,
		build:           cfg.Build,
		artifactPattern: cfg.Artifact,
		client:          client,
	}
}

type jenkinsBuildResponse struct {
	Artifacts []jenkinsArtifact `json:"artifacts"`
}

type jenkinsArtifact struct {
	FileName     string `json:"fileName"`
	RelativePath string `json:"relativePath"`
}

type jenkinsSource struct {
	url             string
	job             string
	build           int
	artifactPattern string
	client          *http.Client
}

func (s *jenkinsSource) resolvedBuildPath() string {
	if s.build > 0 {
		return fmt.Sprintf("%d", s.build)
	}
	return "lastSuccessfulBuild"
}

func (s *jenkinsSource) resolve() (downloadURL, resolvedFileName, buildPath string, err error) {
	baseURL := strings.TrimRight(s.url, "/")
	buildPath = s.resolvedBuildPath()

	apiURL := fmt.Sprintf("%s/job/%s/%s/api/json", baseURL, s.job, buildPath)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", "", err
	}
	applyJenkinsAuth(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("fetching jenkins build info for %s: %w", s.job, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("jenkins API returned status %d for job %s", resp.StatusCode, s.job)
	}

	var buildResp jenkinsBuildResponse
	if err := json.NewDecoder(resp.Body).Decode(&buildResp); err != nil {
		return "", "", "", fmt.Errorf("decoding jenkins response for %s: %w", s.job, err)
	}

	matched, err := matchArtifact(buildResp.Artifacts, s.artifactPattern)
	if err != nil {
		return "", "", "", fmt.Errorf("matching artifact for job %s: %w", s.job, err)
	}

	downloadURL = fmt.Sprintf("%s/job/%s/%s/artifact/%s", baseURL, s.job, buildPath, matched.RelativePath)
	return downloadURL, matched.FileName, buildPath, nil
}

func (s *jenkinsSource) download(client *http.Client, destPath string, onProgress ProgressFunc) error {
	downloadURL, _, _, err := s.resolve()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	applyJenkinsAuth(req)

	return writeHTTPResponse(req, client, destPath, onProgress)
}

func (s *jenkinsSource) cacheKey() string {
	return cacheKey("jenkins", s.url, s.job, s.resolvedBuildPath(), s.artifactPattern)
}

func (s *jenkinsSource) describe() string {
	return fmt.Sprintf("jenkins:%s/%s", s.url, s.job)
}

func (s *jenkinsSource) fileName() (string, error) {
	_, name, _, err := s.resolve()
	return name, err
}

func matchArtifact(artifacts []jenkinsArtifact, pattern string) (*jenkinsArtifact, error) {
	for i := range artifacts {
		matched, err := filepath.Match(pattern, artifacts[i].FileName)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
		}
		if matched {
			return &artifacts[i], nil
		}
	}
	return nil, fmt.Errorf("no artifact matching pattern %q", pattern)
}

func applyJenkinsAuth(req *http.Request) {
	user := os.Getenv("JENKINS_USER")
	token := os.Getenv("JENKINS_TOKEN")
	if user != "" && token != "" {
		req.SetBasicAuth(user, token)
	}
}
