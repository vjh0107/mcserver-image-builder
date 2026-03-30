package config

import "go.junhyung.kr/mcserver-image-builder/internal/schema"

type ServerConfig struct {
	Kind       schema.Kind  `yaml:"kind"`
	Name       string       `yaml:"name"`
	Components []string     `yaml:"components,omitempty"`
	Source     ServerSource  `yaml:"source,omitempty"`
	Warm       *WarmConfig  `yaml:"warm,omitempty"`
	Plugins    []Plugin     `yaml:"plugins"`
	Resources  []Resource   `yaml:"resources"`
}

func (c *ServerConfig) WarmArtifacts() []string {
	var paths []string
	for _, p := range c.Plugins {
		if p.Warm != nil {
			paths = append(paths, p.Warm.Artifacts...)
		}
	}
	return paths
}

type DownloadSource struct {
	URL      string          `yaml:"url,omitempty"`
	Jenkins  *JenkinsSource  `yaml:"jenkins,omitempty"`
	TeamCity *TeamCitySource `yaml:"teamcity,omitempty"`
}

func (s *DownloadSource) sourceCount() int {
	n := 0
	if s.URL != "" {
		n++
	}
	if s.Jenkins != nil {
		n++
	}
	if s.TeamCity != nil {
		n++
	}
	return n
}

func (s *DownloadSource) IsEmpty() bool {
	return s.sourceCount() == 0
}

type ServerSource struct {
	DownloadSource `yaml:",inline"`
	PaperMC        *PaperMCSource `yaml:"papermc,omitempty"`
}

func (s *ServerSource) sourceCount() int {
	n := s.DownloadSource.sourceCount()
	if s.PaperMC != nil {
		n++
	}
	return n
}

func (s *ServerSource) IsEmpty() bool {
	return s.sourceCount() == 0
}

type PaperMCSource struct {
	Project string `yaml:"project,omitempty"`
	Version string `yaml:"version"`
	Build   int    `yaml:"build"`
}

type WarmConfig struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Timeout string `yaml:"timeout,omitempty"`
	Memory  string `yaml:"memory,omitempty"`
}

type Plugin struct {
	Name      string         `yaml:"name"`
	Source    DownloadSource `yaml:"source"`
	Extract   bool           `yaml:"extract,omitempty"`
	Resources []Resource     `yaml:"resources,omitempty"`
	Warm      *PluginWarm    `yaml:"warm,omitempty"`
}

type ResourceSource struct {
	DownloadSource `yaml:",inline"`
	Path           string `yaml:"path,omitempty"`
}

func (s *ResourceSource) sourceCount() int {
	n := s.DownloadSource.sourceCount()
	if s.Path != "" {
		n++
	}
	return n
}

type Resource struct {
	Source    ResourceSource `yaml:"source"`
	MountPath string        `yaml:"mountPath"`
	Extract   bool          `yaml:"extract,omitempty"`
	Stage     string        `yaml:"stage,omitempty"`
}

func (r Resource) IsRemote() bool {
	return r.Source.Path == ""
}

func (r Resource) ForWarm() bool {
	return r.Stage == "" || r.Stage == "all" || r.Stage == "warm"
}

func (r Resource) ForBuild() bool {
	return r.Stage == "" || r.Stage == "all" || r.Stage == "build"
}

type PluginWarm struct {
	Artifacts []string `yaml:"artifacts"`
}

type JenkinsSource struct {
	URL      string `yaml:"url"`
	Job      string `yaml:"job"`
	Build    int    `yaml:"build,omitempty"`
	Artifact string `yaml:"artifact"`
}

type TeamCitySource struct {
	URL       string `yaml:"url"`
	BuildType string `yaml:"buildType"`
	Build     int    `yaml:"build,omitempty"`
	Artifact  string `yaml:"artifact"`
}
