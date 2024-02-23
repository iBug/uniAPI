# Docker plugin

This plugin implements two components:

- `docker.attachexec`: Implements the Commander interface by sending commands through `docker attach` and reading the output.
- `docker.logs`: Implements the Streamer interface by reading the logs of the container. Writes are simply discarded.
- `docker.stream`: Implements the Streamer interface by attaching to the container.

All components share the same config:

```yaml
host: "unix:///var/run/docker.sock"
container: container_name
timeout: 0s
```

Additionally, `docker.logs` supports an extra config `stderr: true` (default `false`) to read the stderr logs instead of the stdout logs.
