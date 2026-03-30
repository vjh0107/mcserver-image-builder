package schema

import "path/filepath"

const (
	KindServer    Kind = "Server"
	KindProxy     Kind = "Proxy"
	KindComponent Kind = "Component"
)

var Default = NewScheme()

func init() {
	Default.Register(KindServer, Profile{
		DefaultProject: "paper",
		DefaultWarm:    true,
		CacheArtifacts: []string{"libraries", "cache", "versions", filepath.Join("plugins", ".paper-remapped")},
		DockerTemplate: "server",
	})

	Default.Register(KindProxy, Profile{
		DefaultProject: "velocity",
		DockerTemplate: "proxy",
	})

	Default.AddKind(KindComponent)
}
