package config

func SetDefaults(cfg *ServerConfig) {
	profile, err := cfg.Kind.Profile()
	if err != nil {
		return
	}

	if cfg.Warm == nil {
		cfg.Warm = &WarmConfig{}
	}
	if cfg.Warm.Enabled == nil {
		v := profile.DefaultWarm
		cfg.Warm.Enabled = &v
	}
	if cfg.Warm.Timeout == "" {
		cfg.Warm.Timeout = "5m"
	}
	if cfg.Warm.Memory == "" {
		cfg.Warm.Memory = "2G"
	}

	src := &cfg.Source
	if src.PaperMC != nil && src.PaperMC.Project == "" {
		src.PaperMC.Project = profile.DefaultProject
	}
}
