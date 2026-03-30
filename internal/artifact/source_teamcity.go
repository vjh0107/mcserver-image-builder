package artifact

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.junhyung.kr/mcserver-image-builder/internal/config"
)

func init() {
	registerJarResolver(resolveTeamCityJar)
	registerDownloadResolver(resolveTeamCityDownload)
}

func resolveTeamCityDownload(src *config.DownloadSource, _ *http.Client) (source, bool) {
	if src.TeamCity == nil {
		return nil, false
	}
	return newTeamCitySource(src.TeamCity), true
}

func resolveTeamCityJar(src *config.ServerSource, client *http.Client) (source, bool) {
	return resolveTeamCityDownload(&src.DownloadSource, client)
}

func newTeamCitySource(cfg *config.TeamCitySource) *teamCitySource {
	return &teamCitySource{
		url:       cfg.URL,
		buildType: cfg.BuildType,
		build:     cfg.Build,
		artifact:  cfg.Artifact,
	}
}

type teamCitySource struct {
	url       string
	buildType string
	build     int
	artifact  string
}

func (s *teamCitySource) resolvedBuildSpec() string {
	if s.build > 0 {
		return fmt.Sprintf("%d.id", s.build)
	}
	return ".lastSuccessful"
}

func (s *teamCitySource) download(client *http.Client, destPath string, onProgress ProgressFunc) error {
	baseURL := strings.TrimRight(s.url, "/")
	buildSpec := s.resolvedBuildSpec()

	downloadURL := fmt.Sprintf("%s/repository/download/%s/%s/%s", baseURL, s.buildType, buildSpec, s.artifact)

	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("TEAMCITY_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return writeHTTPResponse(req, client, destPath, onProgress)
}

func (s *teamCitySource) cacheKey() string {
	return cacheKey("teamcity", s.url, s.buildType, s.resolvedBuildSpec(), s.artifact)
}

func (s *teamCitySource) describe() string {
	return fmt.Sprintf("teamcity:%s/%s", s.url, s.buildType)
}

func (s *teamCitySource) fileName() (string, error) {
	return filepath.Base(s.artifact), nil
}
