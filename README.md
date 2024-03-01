# api-ustc

A custom server that converts many data sources into a versatile API. See the [`plugins/` directory](plugins/) for supported plugins and their configurations.

## Configuration

The configuration file uses the YAML format.

Currently the configuration has one single key `services` at the root level. It is a "key-service" map. The key will be used in URL routing and the request will be served by the defined service.

For example, with the following short configuration:

```yaml
services:
  robots.txt:
    type: robotstxt
  minecraft:
    type: minecraft
    commander:
      type: rcon
      server: 192.0.2.0
      port: 25575
      password: rcon_password
      timeout: 100ms
```

A request to `/robots.txt` will be served by the [`robotstxt` service](plugins/robotstxt/), and a request to `/minecraft` will be served by the [`minecraft` service](plugins/minecraft/). The `minecraft` service will use the [`rcon` Commander](plugins/rcon/) to connect to the actual Minecraft server to retrieve information.

An longer example of configuration:

```yaml
services:
  robots.txt:
    type: robotstxt
  csgo:
    type: csgo
    commander:
      type: rcon
      server: 192.0.2.0
      port: 27015
      password: rcon_password
      timeout: 100ms
    streamer:
      type: docker.streamer
      host: "unix:///var/run/docker.sock"
      container: cs2
  factorio:
    type: factorio
    commander:
      type: rcon
      server: 192.0.2.0
      port: 34197
      password: rcon_password
      timeout: 100ms
  minecraft:
    type: minecraft
    commander:
      type: rcon
      server: 192.0.2.0
      port: 25575
      password: rcon_password
      timeout: 100ms
  palworld:
    type: palworld
    commander:
      type: rcon
      server: 192.0.2.0
      port: 8211
      password: rcon_password
      timeout: 100ms
  teamspeak:
    type: token-protected
    tokens:
      - some_stupid_token
    service:
      type: teamspeak
      key: serverquery_api_key
      instance: "1"
      endpoint: ts.example.com
      timeout: 100ms
  terraria:
    type: terraria
    streamer:
      type: docker.stream
      host: "unix:///var/run/docker.sock"
      container: terraria
  206ip:
    type: wireguard.endpoint
    interface: wg0
    public-key: AAAA==
    use-sudo: true
```

## Classes

These are defined in [`common/interfaces.go`](common/interfaces.go). Some of the classes are:

- **Service**: Provides an HTTP handler for something.

  Additionally, the root server is also a Service (going by the name `server`).
  You can achieve sub-path routing by defining a `server` Service.
  For example:

  ```yaml
  services:
    some_path:
      type: server
      services:
        sub_path:
          type: some_service
  ```

  Then `some_service` will be available at `/some_path/sub_path`.

- **Commander**: Provides a way to execute commands and retrieve the output. For example, many game servers uses the [RCON protocol](https://developer.valvesoftware.com/wiki/Source_RCON_Protocol) as a command interface.
- **Streamer**: Provides a way to interact with a stream of data. For example, sending input to and reading output from a game server console. The [`docker` plugin](plugins/docker/) provides a few Streamers to interact with Docker containers.

A plugin may require another plugin to work. For example, the `minecraft` plugin requires a Commander, but you can use either `rcon` or `docker.attachexec` to interact with a Minecraft server, depending on your setup. The `type` key specifies which plugin to use, and the rest of the config is passed to the plugin.
