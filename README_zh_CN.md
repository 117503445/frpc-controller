# frpc-controller

> 基于 Docker Labels 自动生成 frpc 配置文件

当需要将较多容器通过 frp 实现内网穿透时，手动维护 frpc 配置文件将是一个枯燥的工作。通过 `frpc-controller`，可以将服务的域名前缀和容器端口定义在 Docker Labels 中，并自动生成 frpc 配置文件。

## 快速开始

准备配置文件和 Docker Compose 声明文件

```sh
git clone https://github.com/117503445/frpc-controller.git
cd frpc-controller/docs/example
```

启动服务

```sh
docker compose up -d
```

验证

```sh
curl -H "Host: app1.example.com" 127.0.0.1:80
# show output of app1

curl -H "Host: app2.example.com" http://127.0.0.1:80
# show output of app2
```

对于 Traefik, whoami, traefik-provider-frp 等服务的说明，可以参考 <https://github.com/117503445/traefik-provider-frp.git>。

在这个实例中，使用 frpc-controller 替换了 frpc 服务，并且把 app1 和 app2 的容器端口定义在 Docker Labels 中，frpc-controller 会自动生成 frpc 配置文件，并启动 frpc 服务。

## 配置参考

容器内 `/workspace/config.toml` 是被传递给内置 frpc 的配置文件。

`NETWORK_NAME` 环境变量定义了 Docker 网络的名称，请确保 `frpc-controller` 和待映射容器都在此网络中。默认为 `frpc`。

对于 app1，定义以下 label，则 `frpc-controller` 会新建 app1 连接，并将 app1 的 80 端口穿透给 frps。

```yaml
services:
  app1:
    image: traefik/whoami
    hostname: app1
    restart: unless-stopped
    labels:
      - frpc.app1=80
```

## 实现

`frpc-controller` 通过 Docker API 获取容器列表，将代理信息追加到 `/workspace/config.toml` 中，并启动 frpc 服务。
