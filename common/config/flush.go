package config

import (
	"encoding/json"
	"os"
	"srp-go/common/logger"
)

var (
	OldConfig *Config
	NewConfig *Config

	flushConfigHandleList = make([]func(oldConfig *Config, newConfig *Config), 0)
)

type FlushConfigHandle interface {
	Flush(oldConfig *Config, newConfig *Config)
}

func AddFlushConfigHandle(flushConfigHandle func(oldConfig *Config, newConfig *Config)) {
	flushConfigHandleList = append(flushConfigHandleList, flushConfigHandle)
}

func RefreshConfig() (err error) {
	return RefreshConfigByFile(configFilePath)
}

func RefreshConfigByFile(configFilePath string) (err error) {
	fileConfig := getNewConfig()

	// 判断是否读取默认配置文件
	if configFilePath == "" {
		if _, err := os.Stat(defaultConfigFilePath); err == nil {
			configFilePath = defaultConfigFilePath
		}
	}

	// 判断是否需要去读取配置文件
	if configFilePath != "" {
		err = readIniFile(configFilePath, fileConfig)
		if err != nil {
			log.Error("read config file error", err)
			return
		}
	}
	OldConfig = NewConfig
	NewConfig = &fileConfig

	// 先设置日志
	logger.SetLogLevel(fileConfig.Common.Log.Level)
	if logger.Trace {
		if bytes, e := json.Marshal(fileConfig); e == nil {
			log.Trace("new config", string(bytes))
		}
	}

	// 通知配置更新
	for _, h := range flushConfigHandleList {
		h(OldConfig, NewConfig)
	}
	log.Info("refresh config success")
	return
}
