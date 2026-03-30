package warm

import (
	"crypto"
	_ "crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.junhyung.kr/mcserver-image-builder/internal/docker"
	"go.junhyung.kr/mcserver-image-builder/internal/fsutil"
)

const metadataFile = "metadata.json"

type Options struct {
	Timeout string
	Memory  string
	Output  io.Writer
}

type Runner struct {
	configDir      string
	warmDir        string
	cacheArtifacts []string
	extraArtifacts []string
}

func NewRunner(configDir string, cacheArtifacts, extraArtifacts []string) *Runner {
	warmDir := filepath.Join(configDir, ".warm")
	return &Runner{
		configDir:      configDir,
		warmDir:        warmDir,
		cacheArtifacts: cacheArtifacts,
		extraArtifacts: extraArtifacts,
	}
}

func (r *Runner) allArtifacts() []string {
	return append(r.cacheArtifacts, r.extraArtifacts...)
}

func (r *Runner) WarmDir() string {
	return r.warmDir
}

type Metadata struct {
	ServerJar   string   `json:"server_jar"`
	Plugins     []string `json:"plugins"`
	Fingerprint string   `json:"fingerprint"`
	StartedAt   string   `json:"started_at"`
	CompletedAt string   `json:"completed_at"`
}

func (r *Runner) IsComplete() bool {
	_, err := os.Stat(filepath.Join(r.warmDir, metadataFile))
	return err == nil
}

func (r *Runner) LoadMetadata() (*Metadata, error) {
	data, err := os.ReadFile(filepath.Join(r.warmDir, metadataFile))
	if err != nil {
		return nil, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (r *Runner) IsUpToDate(serverJar string, plugins []string) bool {
	meta, err := r.LoadMetadata()
	if err != nil {
		return false
	}
	return meta.Fingerprint == Fingerprint(serverJar, plugins)
}

func Fingerprint(serverJar string, plugins []string) string {
	h := crypto.SHA256.New()
	h.Write([]byte(serverJar))
	for _, p := range plugins {
		h.Write([]byte{0})
		h.Write([]byte(p))
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func (r *Runner) CopyTo(contextDir string) error {
	for _, artifact := range r.allArtifacts() {
		src := filepath.Join(r.warmDir, artifact)
		if _, err := os.Stat(src); err != nil {
			continue
		}

		dst := filepath.Join(contextDir, artifact)
		if err := fsutil.CopyDir(src, dst); err != nil {
			return fmt.Errorf("copying warm artifact %s: %w", artifact, err)
		}
	}
	return nil
}

func (r *Runner) Clean() error {
	if err := os.RemoveAll(r.warmDir); err != nil {
		return fmt.Errorf("cleaning warm directory: %w", err)
	}
	return nil
}

func (r *Runner) Run(serverJar string, plugins []string, paperJarPath, pluginsDir string, opts Options) error {
	startedAt := time.Now()

	workDir, err := os.MkdirTemp("", "mc-warm-*")
	if err != nil {
		return fmt.Errorf("creating warm work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	if err := r.prepareWorkDir(workDir, paperJarPath, pluginsDir, output); err != nil {
		return err
	}

	if err := r.runServer(workDir, opts); err != nil {
		return err
	}

	stagingDir := r.warmDir + ".staging"
	os.RemoveAll(stagingDir)
	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("creating staging directory: %w", err)
	}

	if err := r.collectArtifactsTo(workDir, stagingDir, output); err != nil {
		os.RemoveAll(stagingDir)
		return err
	}

	meta := Metadata{
		ServerJar:   serverJar,
		Plugins:     plugins,
		Fingerprint: Fingerprint(serverJar, plugins),
		StartedAt:   startedAt.Format(time.RFC3339),
		CompletedAt: time.Now().Format(time.RFC3339),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		os.RemoveAll(stagingDir)
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, metadataFile), metaBytes, 0o644); err != nil {
		os.RemoveAll(stagingDir)
		return fmt.Errorf("writing metadata: %w", err)
	}

	os.RemoveAll(r.warmDir)
	if err := os.Rename(stagingDir, r.warmDir); err != nil {
		return fmt.Errorf("finalizing warm cache: %w", err)
	}

	return nil
}

func (r *Runner) prepareWorkDir(workDir, paperJarPath, pluginsDir string, output io.Writer) error {
	if err := fsutil.CopyFile(paperJarPath, filepath.Join(workDir, "server.jar")); err != nil {
		return fmt.Errorf("copying paper jar: %w", err)
	}

	if err := os.WriteFile(filepath.Join(workDir, "eula.txt"), []byte("eula=true\n"), 0o644); err != nil {
		return fmt.Errorf("writing eula.txt: %w", err)
	}

	if err := fsutil.CopyDir(pluginsDir, filepath.Join(workDir, "plugins")); err != nil {
		return fmt.Errorf("copying plugins: %w", err)
	}

	for _, artifact := range r.allArtifacts() {
		src := filepath.Join(r.warmDir, artifact)
		if _, err := os.Stat(src); err == nil {
			dest := filepath.Join(workDir, artifact)
			fmt.Fprintf(output, "Reusing cached %s\n", artifact)
			if err := fsutil.CopyDir(src, dest); err != nil {
				return fmt.Errorf("copying existing cache %s: %w", artifact, err)
			}
		}
	}

	return nil
}

func (r *Runner) runServer(workDir string, opts Options) error {
	timeout, err := time.ParseDuration(opts.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout %q: %w", opts.Timeout, err)
	}

	hostPort, err := findAvailablePort()
	if err != nil {
		return fmt.Errorf("finding available port: %w", err)
	}

	containerName := fmt.Sprintf("mcserver-warm-%s-%d", filepath.Base(workDir), time.Now().UnixNano())
	memory := opts.Memory

	args := []string{
		"run", "--rm", "--name", containerName,
		"-v", workDir + ":/server",
		"-w", "/server",
		"-p", fmt.Sprintf("%d:25565", hostPort),
		docker.DefaultBaseImage,
		"java", fmt.Sprintf("-Xmx%s", memory), fmt.Sprintf("-Xms%s", memory),
		"-jar", "server.jar",
		"--nogui",
	}

	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	fmt.Fprintf(output, "Starting container (image=%s, memory=%s, port=%d)\n", docker.DefaultBaseImage, memory, hostPort)

	cmd := exec.Command("docker", args...)
	cmd.Stdout = output
	cmd.Stderr = output

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting server container: %w", err)
	}

	defer exec.Command("docker", "rm", "-f", containerName).Run()

	addr := fmt.Sprintf("127.0.0.1:%d", hostPort)
	if err := waitForMinecraftPing(addr, timeout); err != nil {
		cmd.Wait()
		return err
	}

	fmt.Fprintf(output, "Server ready, shutting down\n")
	exec.Command("docker", "stop", "-t", "10", containerName).Run()
	cmd.Wait()

	return nil
}

func waitForMinecraftPing(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := pingMinecraft(addr); err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("server startup timed out after %s", timeout)
}

func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

func (r *Runner) collectArtifactsTo(workDir, destDir string, output io.Writer) error {
	for _, artifact := range r.allArtifacts() {
		src := filepath.Join(workDir, artifact)
		if _, err := os.Stat(src); err != nil {
			continue
		}

		dest := filepath.Join(destDir, artifact)
		if err := fsutil.CopyDir(src, dest); err != nil {
			return fmt.Errorf("collecting cache artifact %s: %w", artifact, err)
		}

		fmt.Fprintf(output, "Collected %s\n", artifact)
	}

	return nil
}
