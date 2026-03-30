package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"go.junhyung.kr/mcserver-image-builder/internal/config"
	"go.junhyung.kr/mcserver-image-builder/internal/docker"
	"go.junhyung.kr/mcserver-image-builder/internal/fsutil"
	"go.junhyung.kr/mcserver-image-builder/internal/ui"
	"go.junhyung.kr/mcserver-image-builder/internal/warm"
)

type buildOptions struct {
	file        string
	tag         string
	registry    string
	baseImage   string
	all         bool
	noWarm      bool
	noCache     bool
	push        bool
	concurrency int
}

func NewBuildCommand() *cobra.Command {
	opts := &buildOptions{}

	cmd := &cobra.Command{
		Use:   "build [server-name]",
		Short: "Build a Minecraft server Docker image",
		Long: `Build a Docker image from a mcserver.yaml configuration.

Discovers server configurations by name from the working directory,
downloads the server JAR, plugins, and resources, then builds the image.

Examples:
  mcserver build lobby
  mcserver build --all --registry registry.example.com/smp
  mcserver build -f path/to/mcserver.yaml -t my-server:latest`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: serverNameCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.all {
				return runBuildAll(cmd, opts)
			}
			return runBuild(cmd, opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "path to mcserver.yaml")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "image tag")
	cmd.Flags().StringVar(&opts.registry, "registry", "", "registry prefix for auto-generated tag")
	cmd.Flags().StringVar(&opts.baseImage, "base-image", "", "base Docker image (default: "+docker.DefaultBaseImage+")")
	cmd.Flags().BoolVar(&opts.all, "all", false, "build all servers in workspace")
	cmd.Flags().BoolVar(&opts.noWarm, "no-warm", false, "allow building without warm cache")
	cmd.Flags().BoolVar(&opts.noCache, "no-cache", false, "ignore all caches and rebuild from scratch")
	cmd.Flags().BoolVar(&opts.push, "push", false, "push image after build")
	cmd.Flags().IntVar(&opts.concurrency, "concurrency", 5, "max concurrent artifact downloads")

	return cmd
}

func runBuild(cmd *cobra.Command, opts *buildOptions, args []string) error {
	if err := docker.EnsureAvailable(); err != nil {
		return err
	}

	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	configPath, err := resolveConfig(opts.file, args, workingDir)
	if err != nil {
		return err
	}

	cfg, err := config.LoadWithComponents(configPath, workingDir)
	if err != nil {
		return err
	}

	tag := resolveTag(opts.tag, opts.registry, cfg.Name)
	label := fmt.Sprintf("%s → %s", cfg.Name, tag)
	stepNames := buildStepNames(cfg, opts.push)

	return ui.RunBuild(label, stepNames, func(n ui.Notifier) error {
		return doBuild(n, cfg, configPath, workingDir, tag, opts.baseImage, opts.noWarm, opts.noCache, opts.push, opts.concurrency)
	})
}

func runBuildAll(cmd *cobra.Command, opts *buildOptions) error {
	if err := docker.EnsureAvailable(); err != nil {
		return err
	}

	workingDir, err := resolveWorkingDir(cmd)
	if err != nil {
		return err
	}

	entries, err := Discover(workingDir)
	if err != nil {
		return err
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	ui.List("Discovered servers", names)

	var configs []*config.ServerConfig
	var paths []string
	for _, entry := range entries {
		cfg, err := config.LoadWithComponents(entry.FilePath, workingDir)
		if err != nil {
			return fmt.Errorf("loading %s: %w", entry.Name, err)
		}
		configs = append(configs, cfg)
		paths = append(paths, entry.FilePath)
	}

	var buildEntries []ui.BuildEntry
	for i, cfg := range configs {
		tag := resolveTag("", opts.registry, cfg.Name)
		label := fmt.Sprintf("%s → %s", cfg.Name, tag)
		stepNames := buildStepNames(cfg, opts.push)

		cfgCopy := cfg
		pathCopy := paths[i]
		tagCopy := tag

		buildEntries = append(buildEntries, ui.BuildEntry{
			Label:     label,
			StepNames: stepNames,
			BuildFn: func(n ui.Notifier) error {
				return doBuild(n, cfgCopy, pathCopy, workingDir, tagCopy, opts.baseImage, opts.noWarm, opts.noCache, opts.push, opts.concurrency)
			},
		})
	}

	return ui.RunParallelBuild(buildEntries)
}

func buildStepNames(cfg *config.ServerConfig, push bool) []string {
	steps := []string{"Resolving artifacts"}
	if *cfg.Warm.Enabled {
		steps = append(steps, "Warm cache")
	}
	steps = append(steps, "Docker build")
	if push {
		steps = append(steps, "Push")
	}
	return steps
}

func doBuild(n ui.Notifier, cfg *config.ServerConfig, configPath, workingDir, tag, baseImage string, noWarm, noCache, push bool, concurrency int) error {
	totalStart := time.Now()
	configDir := filepath.Dir(configPath)

	contextBuilder := docker.NewContextBuilder(cfg.Name)
	defer contextBuilder.Cleanup()

	contextDir, err := contextBuilder.Prepare()
	if err != nil {
		return err
	}

	provider, err := newProvider(noCache)
	if err != nil {
		return err
	}

	// Step 0: Resolving artifacts
	step := 0
	n.Start(step)
	start := time.Now()

	type artifactJob struct {
		name string
		fn   func() error
	}

	jarDesc := provider.ResolveJarDescription(&cfg.Source)
	var jobs []artifactJob

	jobs = append(jobs, artifactJob{
		name: "server.jar",
		fn: func() error {
			return provider.FetchServerJarWithProgress(&cfg.Source, contextDir, func(received, total int64) {
				n.ArtifactProgress("server.jar", received, total)
			})
		},
	})

	for _, p := range cfg.Plugins {
		p := p
		jobs = append(jobs, artifactJob{
			name: p.Name,
			fn: func() error {
				return provider.FetchPluginWithProgress(p, contextDir, func(received, total int64) {
					n.ArtifactProgress(p.Name, received, total)
				})
			},
		})
	}

	for _, r := range cfg.Resources {
		r := r
		name := r.Source.Path
		if r.IsRemote() {
			name = r.MountPath
		}
		rName := name
		jobs = append(jobs, artifactJob{
			name: rName,
			fn: func() error {
				if r.IsRemote() {
					dest := filepath.Join(contextDir, r.MountPath)
					return provider.FetchDownloadWithProgress(&r.Source.DownloadSource, dest, r.Extract, func(received, total int64) {
						n.ArtifactProgress(rName, received, total)
					})
				}
				path := r.Source.Path
				src := resolveSrcPath(path, workingDir, configDir)
				dst := filepath.Join(contextDir, path)
				if strings.HasPrefix(path, "./") {
					dst = filepath.Join(contextDir, path[2:])
				}
				return fsutil.Copy(src, dst)
			},
		})
	}

	// Start all artifacts
	for _, job := range jobs {
		n.ArtifactStart(job.name)
	}

	// Run with semaphore
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, job := range jobs {
		wg.Add(1)
		go func(j artifactJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mu.Lock()
			if firstErr != nil {
				mu.Unlock()
				return
			}
			mu.Unlock()

			if err := j.fn(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", j.name, err)
				}
				mu.Unlock()
				return
			}

			detail := ""
			if j.name == "server.jar" {
				detail = jarDesc
			}
			n.ArtifactDone(j.name, detail)
		}(job)
	}
	wg.Wait()

	if firstErr != nil {
		n.Error(step, firstErr)
		return firstErr
	}

	if err := copyPluginResources(workingDir, configDir, cfg.Plugins, contextDir, "build"); err != nil {
		n.Error(step, err)
		return err
	}

	elapsed := time.Since(start)
	n.Done(step, fmt.Sprintf("%.1fs", elapsed.Seconds()))

	// Warm cache
	hasWarmCache := false
	if *cfg.Warm.Enabled {
		step++
		n.Start(step)
		warmStart := time.Now()

		profile, _ := cfg.Kind.Profile()
		warmRunner := warm.NewRunner(configDir, profile.CacheArtifacts, cfg.WarmArtifacts())
		pluginNames := pluginNameList(cfg.Plugins)

		if !noCache && warmRunner.IsUpToDate(jarDesc, pluginNames) {
			n.Done(step, "up to date")
			hasWarmCache = true
		} else if noWarm {
			n.Done(step, "skipped (--no-warm)")
		} else {
			if err := executeWarm(workingDir, cfg, configPath, noCache, n.LogWriter()); err != nil {
				n.Error(step, err)
				return err
			}
			n.Done(step, fmt.Sprintf("%.1fs", time.Since(warmStart).Seconds()))
			hasWarmCache = true
		}

		if hasWarmCache {
			if err := warmRunner.CopyTo(contextDir); err != nil {
				n.Error(step, err)
				return err
			}
		}
	}

	if baseImage == "" {
		baseImage = docker.DefaultBaseImage
	}

	templateData := docker.TemplateData{
		Kind:      cfg.Kind,
		BaseImage: baseImage,
		WarmCache: hasWarmCache,
		Files:     toDockerFiles(cfg.Resources),
	}

	if err := docker.RenderDockerfile(templateData, contextDir); err != nil {
		return err
	}

	// Docker build
	step++
	n.Start(step)
	buildStart := time.Now()
	buildOpts := docker.BuildOptions{
		ContextDir: contextDir,
		Tag:        tag,
		NoCache:    noCache,
		Output:     n.LogWriter(),
	}
	if err := docker.Build(buildOpts); err != nil {
		n.Error(step, err)
		return err
	}
	n.Done(step, fmt.Sprintf("(%s) %.1fs", tag, time.Since(buildStart).Seconds()))

	// Push
	if push {
		step++
		n.Start(step)
		pushStart := time.Now()
		if err := docker.Push(tag); err != nil {
			n.Error(step, err)
			return err
		}
		n.Done(step, fmt.Sprintf("%.1fs", time.Since(pushStart).Seconds()))
	}

	n.Elapsed(fmt.Sprintf("%.1fs", time.Since(totalStart).Seconds()))
	return nil
}
