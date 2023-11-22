package main

import (
	"flag"
	"simpledns/internal/config"
	"simpledns/internal/dnsserver"
	"simpledns/internal/logger"
)

func main() {
	initServices()
	dnsServer := dnsserver.New(config.Cfg)

	if err := dnsServer.Listen(); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to start DNS server")
	}

}

func initServices() {
	configPathFlag := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	logger.Init()
	config.Init(*configPathFlag)
	dnsserver.Init()
}
