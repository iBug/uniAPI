# Docker plugin

This plugin implements two components:

- `docker.execattach`: Implements the Commander interface by sending commands through `docker attach` and reading the output.
- `docker.stream`: Implements the Streamer interface by attaching to the container.

All components share the same config:

```yaml
host: "unix:///var/run/docker.sock"
container: container_name
```
