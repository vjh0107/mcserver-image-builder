package config

import (
	"fmt"
	"time"
)

func Validate(cfg *ServerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}

	if _, err := cfg.Kind.Profile(); err != nil {
		return err
	}

	if err := validateServerSource(&cfg.Source); err != nil {
		return fmt.Errorf("source: %w", err)
	}

	for i, r := range cfg.Resources {
		if r.Source.sourceCount() == 0 {
			return fmt.Errorf("resources[%d].source: one of path, url, jenkins, or teamcity is required", i)
		}
		if r.Source.sourceCount() > 1 {
			return fmt.Errorf("resources[%d].source: path, url, jenkins, and teamcity are mutually exclusive", i)
		}
		if r.MountPath == "" {
			return fmt.Errorf("resources[%d]: mountPath is required", i)
		}
		if r.IsRemote() {
			if err := validateCISource(r.Source.Jenkins, r.Source.TeamCity, fmt.Sprintf("resources[%d].source", i)); err != nil {
				return err
			}
		}
		if r.Stage != "" && r.Stage != "all" && r.Stage != "build" && r.Stage != "warm" {
			return fmt.Errorf("resources[%d]: stage must be all, build, or warm", i)
		}
	}

	for i, p := range cfg.Plugins {
		if p.Name == "" {
			return fmt.Errorf("plugins[%d].name is required", i)
		}
		prefix := fmt.Sprintf("plugins[%d] (%s)", i, p.Name)
		if err := validateDownloadSource(&p.Source, prefix); err != nil {
			return err
		}
		for j, f := range p.Resources {
			if f.Source.sourceCount() == 0 {
				return fmt.Errorf("%s: resources[%d].source is required", prefix, j)
			}
			if f.MountPath == "" {
				return fmt.Errorf("%s: resources[%d].mountPath is required", prefix, j)
			}
			if f.Stage != "" && f.Stage != "all" && f.Stage != "build" && f.Stage != "warm" {
				return fmt.Errorf("%s: resources[%d].stage must be all, build, or warm", prefix, j)
			}
		}
	}

	if cfg.Warm != nil && cfg.Warm.Timeout != "" {
		if _, err := time.ParseDuration(cfg.Warm.Timeout); err != nil {
			return fmt.Errorf("warm.timeout: invalid duration %q", cfg.Warm.Timeout)
		}
	}

	return nil
}

func validateDownloadSource(src *DownloadSource, prefix string) error {
	if src.sourceCount() == 0 {
		return fmt.Errorf("%s: one of url, jenkins, or teamcity is required", prefix)
	}
	if src.sourceCount() > 1 {
		return fmt.Errorf("%s: url, jenkins, and teamcity are mutually exclusive", prefix)
	}
	return validateCISource(src.Jenkins, src.TeamCity, prefix)
}

func validateServerSource(src *ServerSource) error {
	if src.sourceCount() == 0 {
		return fmt.Errorf("one of url, papermc, jenkins, or teamcity is required")
	}
	if src.sourceCount() > 1 {
		return fmt.Errorf("url, papermc, jenkins, and teamcity are mutually exclusive")
	}

	if src.PaperMC != nil {
		if src.PaperMC.Version == "" {
			return fmt.Errorf("papermc.version is required")
		}
		if src.PaperMC.Build == 0 {
			return fmt.Errorf("papermc.build is required")
		}
	}

	return validateCISource(src.Jenkins, src.TeamCity, "source")
}

func validateCISource(jenkins *JenkinsSource, teamcity *TeamCitySource, prefix string) error {
	if jenkins != nil {
		if jenkins.URL == "" {
			return fmt.Errorf("%s: jenkins.url is required", prefix)
		}
		if jenkins.Job == "" {
			return fmt.Errorf("%s: jenkins.job is required", prefix)
		}
		if jenkins.Artifact == "" {
			return fmt.Errorf("%s: jenkins.artifact is required", prefix)
		}
	}
	if teamcity != nil {
		if teamcity.URL == "" {
			return fmt.Errorf("%s: teamcity.url is required", prefix)
		}
		if teamcity.BuildType == "" {
			return fmt.Errorf("%s: teamcity.buildType is required", prefix)
		}
		if teamcity.Artifact == "" {
			return fmt.Errorf("%s: teamcity.artifact is required", prefix)
		}
	}
	return nil
}
