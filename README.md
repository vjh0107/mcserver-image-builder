# Cloud Native Docker Image Builder for Minecraft Servers

Write a mcserver.yaml, run mcserver build, get a production-ready Docker image with your server JAR, plugins, configs, and warm caches baked in.

## Why

If you've ever containerized a Minecraft server, you've probably written Dockerfiles that curl JARs, copy plugin configs by hand, and break when you bump a version. Updating Paper or a plugin means editing shell scripts and hoping nothing drifts. Or more realistically, you just gave up and ran it on bare metal.

This project replaces all of that with declarative YAML configs. Declare what you want — server version, plugins, configs — and the tool handles downloading, caching, Dockerfile generation, and image building.

Inspired by Kustomize, it also supports composable components for managing shared configs across multiple servers.

## How it compares to itzg/docker-minecraft-server

[itzg/docker-minecraft-server](https://github.com/itzg/docker-minecraft-server) is a well-maintained project by an experienced developer, and it works great for most use cases. This is not a replacement — it's a different approach.

itzg provides one general-purpose image. The container downloads the server JAR, installs plugins, and configures itself at startup using environment variables. Simple and flexible.

This project builds a dedicated image per server. The server JAR, plugins, configs, and warm caches are all baked into the image at build time. Containers start immediately with zero runtime downloads and no external dependencies.

If you're running multiple servers on Kubernetes and care about startup time, reproducibility, and image immutability, this is the cloud-native approach.

## Documentation

See the [Wiki](https://github.com/vjh0107/mcserver-image-builder/wiki) for full documentation:

- [Getting Started](https://github.com/vjh0107/mcserver-image-builder/wiki/Getting-Started)
- [Configuration](https://github.com/vjh0107/mcserver-image-builder/wiki/Configuration)
- [Sources](https://github.com/vjh0107/mcserver-image-builder/wiki/Sources) — PaperMC, Jenkins, TeamCity, URL
- [Components](https://github.com/vjh0107/mcserver-image-builder/wiki/Components)
- [Warm Cache](https://github.com/vjh0107/mcserver-image-builder/wiki/Warm-Cache)
- [Runtime Environment](https://github.com/vjh0107/mcserver-image-builder/wiki/Runtime-Environment)
- [Examples](https://github.com/vjh0107/mcserver-image-builder/wiki/Examples)

## License

MIT
