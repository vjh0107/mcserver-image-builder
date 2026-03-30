package schema

import "fmt"

type Kind string

func (k Kind) String() string {
	return string(k)
}

func (k Kind) IsValid() bool {
	return Default.IsValid(k)
}

func (k Kind) Profile() (Profile, error) {
	return Default.Profile(k)
}

type Profile struct {
	DefaultProject string
	DefaultWarm    bool
	CacheArtifacts []string
	DockerTemplate string
}

type Scheme struct {
	kinds    map[Kind]bool
	profiles map[Kind]Profile
}

func NewScheme() *Scheme {
	return &Scheme{
		kinds:    make(map[Kind]bool),
		profiles: make(map[Kind]Profile),
	}
}

func (s *Scheme) AddKind(kind Kind) {
	s.kinds[kind] = true
}

func (s *Scheme) Register(kind Kind, profile Profile) {
	s.kinds[kind] = true
	s.profiles[kind] = profile
}

func (s *Scheme) IsValid(kind Kind) bool {
	return s.kinds[kind]
}

func (s *Scheme) Profile(kind Kind) (Profile, error) {
	p, ok := s.profiles[kind]
	if !ok {
		return Profile{}, fmt.Errorf("unknown kind %q", kind)
	}
	return p, nil
}
