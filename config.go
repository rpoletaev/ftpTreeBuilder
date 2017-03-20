package ftpTreeBuilder

import (
	"log"
)

const (
	defaultMaxFTPCons            = 10
	defaultReconnectIterCount    = 5
	defaultReconnectSleepSeconds = 10
)

// FTPBuilderConfig конфиг для постройки дерева по образу ftp
type FTPBuilderConfig struct {
	*log.Logger
	MaxFTPCons            int
	ReconnectIterCount    int
	ReconnectSleepSeconds int
	RootNodeDirectory     string
	FTPAddr               string //"ftp.zakupki.gov.ru:21"
	FTPLogin              string
	FTPPass               string
	DBConString           string
}

// Prepare установка, при необходимости, дефолтных значений
func (cf *FTPBuilderConfig) Prepare() {
	if cf.MaxFTPCons == 0 {
		cf.MaxFTPCons = defaultMaxFTPCons
	}

	if cf.ReconnectIterCount == 0 {
		cf.ReconnectIterCount = defaultReconnectIterCount
	}

	if cf.ReconnectSleepSeconds == 0 {
		cf.ReconnectSleepSeconds = defaultReconnectSleepSeconds
	}
}
