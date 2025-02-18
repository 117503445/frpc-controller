# frpc-controller

> Automatically generate frpc configuration files based on Docker Labels

When you need to expose multiple containers through frp for intranet penetration, manually maintaining the frpc configuration files can be a tedious task. With `frpc-controller`, you can define the service's domain prefix and container port in Docker Labels, and it will automatically generate the frpc configuration files.

## Quick Start

Prepare the configuration files and Docker Compose declaration file.

```sh
git clone https://github.com/117503445/frpc-controller.git
cd frpc-controller/docs/example
```

Start the services.

```sh
docker compose up -d
```

Verify.

```sh
curl -H "Host: app1.example.com" 127.0.0.1:80
# show output of app1

curl -H "Host: app2.example.com" http://127.0.0.1:80
# show output of app2
```

For explanations of services like Traefik, whoami, and traefik-provider-frp, you can refer to <https://github.com/117503445/traefik-provider-frp.git>.

In this example, frpc-controller is used to replace the frpc service, and the container ports for app1 and app2 are defined in Docker Labels. frpc-controller will automatically generate the frpc configuration files and start the frpc service.

## Configuration Reference

The `/workspace/config.toml` file inside the container is the configuration file passed to the built-in frpc.

The `NETWORK_NAME` environment variable defines the name of the Docker network. Ensure that `frpc-controller` and the containers to be mapped are in this network. The default is `frpc`.

For app1, define the following label, and `frpc-controller` will create a new app1 connection and expose app1's port 80 to frps.

```yaml
services:
  app1:
    image: traefik/whoami
    hostname: app1
    restart: unless-stopped
    labels:
      - frpc.app1=80
```

## Implementation

`frpc-controller` retrieves the list of containers via the Docker API, appends the proxy information to `/workspace/config.toml`, and starts the frpc service.
