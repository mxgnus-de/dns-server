package dnsserver

import (
	"fmt"
	"net"
	"simpledns/internal/config"
	"simpledns/internal/logger"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

type domainLookup struct {
	domain string
	ttl    uint32
	class  dns.Class
}

type domainIpLookup struct {
	domainLookup
	ip net.IP
}

type domainCNAMELookup struct {
	domainLookup
	cname string
}

type reverseDomainLookup struct {
	domainLookup
}

type DNSServer struct {
	server              *dns.Server
	domainIPLoopup      map[string]domainIpLookup
	domainCNAMELookup   map[string]domainCNAMELookup
	reverseDomainLookup map[string]reverseDomainLookup
}

var serviceLogger zerolog.Logger

func Init() {
	serviceLogger = logger.CreateService("dnsserver")
}

func New(cfg *config.Config) *DNSServer {
	dnsServer := &DNSServer{}
	if err := dnsServer.applyConfig(cfg); err != nil {
		serviceLogger.Fatal().Err(err).Msg("Failed to apply config")
	}

	return dnsServer
}

func (s *DNSServer) applyConfig(cfg *config.Config) error {
	s.domainIPLoopup = make(map[string]domainIpLookup)
	s.domainCNAMELookup = make(map[string]domainCNAMELookup)
	s.reverseDomainLookup = make(map[string]reverseDomainLookup)
	for _, record := range cfg.Records {
		domainLookup := domainLookup{
			domain: record.Name,
			ttl:    record.TTL,
			class:  record.Class,
		}

		switch record.Type {
		case config.A, config.AAAA:
			ip := net.ParseIP(record.Value)
			if ip == nil {
				return fmt.Errorf("invalid IP address: %s", record.Value)
			}
			s.domainIPLoopup[record.Name] = domainIpLookup{
				domainLookup: domainLookup,
				ip:           ip,
			}

			ipStr := ip.String()
			reversedIpStr := ""
			for i := len(ipStr) - 1; i >= 0; i-- {
				reversedIpStr += string(ipStr[i])
			}

			reservedDomainName := fmt.Sprintf("%s.in-addr.arpa.", reversedIpStr)
			s.reverseDomainLookup[reservedDomainName] = reverseDomainLookup{domainLookup: domainLookup}

			var recordType string
			if record.Type == config.A {
				recordType = "A"
			} else {
				recordType = "AAAA"
			}

			serviceLogger.Debug().Str("domain", domainLookup.domain).Str("ip", ipStr).Str("reversed_domain", reservedDomainName).Str("type", recordType).Msgf("Added %s domain to lookup", recordType)
		case config.CNAME:
			s.domainCNAMELookup[record.Name] = domainCNAMELookup{
				domainLookup: domainLookup,
				cname:        record.Value,
			}
			serviceLogger.Debug().Str("domain", domainLookup.domain).Str("cname", record.Value).Msg("Added CNAME domain to lookup")
		}
	}

	serviceLogger.Info().Int("records", len(s.domainIPLoopup)+len(s.domainCNAMELookup)).Int("reverse_records", len(s.reverseDomainLookup)).Msg("Config applied")

	return nil
}

func (s *DNSServer) Listen() error {
	server := &dns.Server{
		Addr:    ":53",
		Net:     "udp",
		Handler: s,
		UDPSize: 65535,
	}

	s.server = server
	serviceLogger.Info().Str("addr", server.Addr).Str("net", server.Net).Msg("Starting DNS server")
	if err := server.ListenAndServe(); err != nil {
		return err
	}

	return nil
}
