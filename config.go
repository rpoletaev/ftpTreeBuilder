package ftpTreeBuilder

import (
	"log"
)

const (
	defaultMaxFTPCons            = 5
	defaultReconnectIterCount    = 5
	defaultReconnectSleepSeconds = 10
)

// FTPBuilderConfig конфиг для постройки дерева по образу ftp
type FTPBuilderConfig struct {
	*log.Logger
	MaxFTPCons        int    `yaml:"max_ftp_connections"`
	ServicePort       string `yaml:"service_port"`
	RootNodeDirectory string `yaml:"root_node_directory"`
	FTPAddr           string `yaml:"ftp_addr"`
	FTPLogin          string `yaml:"ftp_login"`
	FTPPass           string `yaml:"ftp_pass"`
	DBConString       string `yaml:"db_con_str"`
	RedisConString    string `yaml:"redis_con_str"`
	RedisPassword     string `yaml:"redis_password"`
}

// Prepare установка, при необходимости, дефолтных значений
func (cf *FTPBuilderConfig) Prepare() {
	if cf.MaxFTPCons == 0 {
		cf.MaxFTPCons = defaultMaxFTPCons
	}

	// if cf.ReconnectIterCount == 0 {
	// 	cf.ReconnectIterCount = defaultReconnectIterCount
	// }

	// if cf.ReconnectSleepSeconds == 0 {
	// 	cf.ReconnectSleepSeconds = defaultReconnectSleepSeconds
	// }
}
