package config

import (
	"flag"
	"math/rand"
	"os"
	"srp-go/common/logger"
	"time"
)

var (
	log = logger.NewLog("Config")

	configFilePath        = "" // 配置文件路径
	defaultConfigFilePath = "config.ini"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	// 尝试从参数获取配置文件路径
	flag.StringVar(&configFilePath, "c", configFilePath, "config file path")
	flag.Parse()

	// 初始化
	AddFlushConfigHandle(flushSecurity)

	// 刷新配置
	if err := RefreshConfig(); err != nil {
		os.Exit(1) // 首次启动如果配置文件加载失败则退出
	}
}

type Config struct {
	Common *CommonConfig           `json:"common"`
	P2p    map[string]*P2pConfig   `json:"p2p"`
	Proxy  map[string]*ProxyConfig `json:"proxy"`
	Nat    map[string]*NatConfig   `json:"nat"`
}

func getNewConfig() Config {
	result := Config{&CommonConfig{
		&ListenConfig{"", 13520},
		&ServerConfig{"0.0.0.0", 13520},
		&ClientConfig{},
		&LogConfig{"info"},
		&SecurityConfig{false, "", ""},
		&KeepAliveConfig{30, 10},
	},
		make(map[string]*P2pConfig),
		make(map[string]*ProxyConfig),
		make(map[string]*NatConfig),
	}
	return result
}

type CommonConfig struct {
	Listen    *ListenConfig    `json:"listen"`
	Server    *ServerConfig    `json:"server"`
	Client    *ClientConfig    `json:"client"`
	Log       *LogConfig       `json:"log"`
	Security  *SecurityConfig  `json:"security"`
	KeepAlive *KeepAliveConfig `json:"keep_alive"`
}

type ListenConfig struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

type ServerConfig struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

type ClientConfig struct {
	Name string `json:"name"`
}

type LogConfig struct {
	Level string `json:"level"`
}

type SecurityConfig struct {
	Enable     bool   `json:"enable"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

type KeepAliveConfig struct {
	Interval int `json:"interval"`
	Timeout  int `json:"timeout"`
}

type P2pConfig struct {
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"local_port"`
	TargetName string `json:"target_name"`
	TargetPort int    `json:"target_port"`
}

type ProxyConfig struct {
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"local_port"`
	TargetName string `json:"target_name"`
	TargetIp   string `json:"target_ip"`
	TargetPort int    `json:"target_port"`
}

type NatConfig struct {
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"local_port"`
	TargetName string `json:"target_name"`
	TargetIp   string `json:"target_ip"`
	RemotePort int    `json:"remote_port"`
}
