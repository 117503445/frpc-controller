package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/117503445/goutils"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

func mapsEqual(a, b map[string]target) bool {
	// 如果两个 map 都为 nil，则它们被认为是相等的。
	if a == nil && b == nil {
		return true
	}

	// 如果其中一个为 nil 或者长度不一致，则认为它们不相等。
	if a == nil || b == nil || len(a) != len(b) {
		return false
	}

	// 检查 a 中的每个元素是否在 b 中也存在且值相同。
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}

	// 注意：理论上还需要检查 b 中是否有不在 a 中的额外键，但由于我们之前已经确认了两者的长度相同，
	// 如果所有 a 中的键都存在于 b 中且对应的值相同，那么可以保证两个 map 相等。

	return true
}

type mapping map[string]target

type target struct {
	Ip   string
	Port string
}

type DockerWatcher struct {
	cli  *client.Client
	ch   chan mapping
	last mapping
}

func NewDockerWatcher(ch chan mapping) *DockerWatcher {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal().Err(err).Msg("new docker client error")
	}
	return &DockerWatcher{cli: cli, ch: ch, last: make(mapping)}
}

func (d *DockerWatcher) Start() {
	for {
		cur := make(mapping)

		containers, err := d.cli.ContainerList(context.Background(), container.ListOptions{All: true})
		if err != nil {
			log.Fatal().Err(err).Msg("list docker container error")
		}

		for _, container := range containers {
			inspect, err := d.cli.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				log.Fatal().Err(err).Msg("inspect docker container error")
			}

			ip := ""
			for network, netInfo := range inspect.NetworkSettings.Networks {
				// fmt.Printf("Network: %s, IP Address: %s\n", network, netInfo.IPAddress)
				if network == networkName {
					ip = netInfo.IPAddress
					break
				}
			}
			if ip == "" {
				continue
			}
			for k, v := range inspect.Config.Labels {
				if strings.HasPrefix(k, "frpc") {
					domain := strings.TrimLeft(k, "frpc.")
					port := v
					cur[domain] = target{Ip: ip, Port: port}
				}
			}
		}
		// log.Info().Interface("cur", cur).Msg("")

		if !mapsEqual(cur, d.last) {
			d.ch <- cur
			d.last = cur
		}

		time.Sleep(time.Second * 5)
	}
}

const fileCfg = "./config.toml"
const fileGenCfg = "./config.gen.toml"

type Executor struct {
	cmd     *exec.Cmd
	cmdLock sync.Mutex
}

func NewExecutor() *Executor {
	if !goutils.FileExists(fileCfg) {
		log.Fatal().Msg("config file not found")
	}

	return &Executor{}
}

func (e *Executor) UpdateCfg(m mapping) {
	text, err := goutils.ReadText(fileCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("read config file error")
	}
	if len(m) > 0 {
		text += "\n### generated by frpc-controller ###\n"

		for domain, target := range m {
			text += fmt.Sprintf(`[[proxies]]
name = "%v"
type = "tcp"
localIP = "%v"
localPort = %v
metadatas.domain = ""
	
	`, domain, target.Ip, target.Port)
		}
	}

	log.Debug().Str("text", text).Msg("")

	err = goutils.WriteText(fileGenCfg, text)
	if err != nil {
		log.Fatal().Err(err).Msg("write config file error")
	}
}

func (e *Executor) Start() {
	log.Info().Msg("start frpc")

	e.cmdLock.Lock()
	defer e.cmdLock.Unlock()

	if e.cmd != nil {
		err := e.cmd.Process.Kill()
		if err != nil {
			log.Fatal().Err(err).Msg("kill frpc error")
		}
	}

	cmds := []string{"/usr/bin/frpc", "-c", fileGenCfg}
	e.cmd = exec.Command(cmds[0], cmds[1:]...)
	e.cmd.Stdout = os.Stdout
	e.cmd.Stderr = os.Stdout
	go func() {
		if err := e.cmd.Run(); err != nil && err.Error() != "signal: killed" {
			log.Warn().Err(err).Msg("frpc error")
		}
	}()
}

// networkName is the name of docker network which frpc-controller and other containers are in.
var networkName string

func main() {
	goutils.InitZeroLog()

	networkName = os.Getenv("NETWORK_NAME")
	if networkName == "" {
		networkName = "frp"
		log.Info().Msg("NETWORK_NAME not set, use default network name `frp`")
	} else {
		log.Info().Str("networkName", networkName).Send()
	}

	ch := make(chan mapping)
	watcher := NewDockerWatcher(ch)
	go func() {
		watcher.Start()
	}()

	executor := NewExecutor()
	executor.UpdateCfg(make(mapping))
	executor.Start()

	go func() {
		for {
			m := <-ch
			executor.UpdateCfg(m)
			executor.Start()
		}
	}()

	select {}
}
