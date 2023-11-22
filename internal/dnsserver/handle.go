package dnsserver

import (
	"net"

	"github.com/miekg/dns"
)

func (s *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetReply(r)

	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range r.Question {
			switch q.Qtype {
			case dns.TypeA:
				if rrs, _ := s.resolveARequest(q, true); rrs != nil {
					m.Answer = append(m.Answer, rrs...)
				}
			case dns.TypeAAAA:
				if rrs, _ := s.resolveAAAARequest(q, true); rrs != nil {
					m.Answer = append(m.Answer, rrs...)
				}
			case dns.TypeCNAME:
				if rrs, _ := s.resolveCNAMERequest(q); rrs != nil {
					m.Answer = append(m.Answer, rrs...)
				}
			case dns.TypePTR:
				if rrs, _ := s.resolveReverseRequest(q); rrs != nil {
					m.Answer = append(m.Answer, rrs...)
				}
			default:
				serviceLogger.Debug().Uint16("type", q.Qtype).Msg("Unsupported DNS request type")
				forwardResponse, err := resolve(q)
				if err == nil {
					m.Answer = append(m.Answer, forwardResponse...)
				}
			}
		}

		types := make([]uint16, len(r.Question))
		for i, q := range r.Question {
			types[i] = q.Qtype
		}

		names := make([]string, len(m.Answer))
		for i, a := range m.Answer {
			names[i] = a.Header().Name
		}

		if len(m.Answer) == 0 && len(r.Question) > 0 {
			names := make([]string, len(r.Question))
			for i, q := range r.Question {
				names[i] = q.Name
			}

			serviceLogger.Debug().Interface("names", names).Interface("types", types).Msg("DNS response not found")
			m.SetRcode(r, dns.RcodeNameError)
		} else {
			serviceLogger.Debug().Interface("names", names).Interface("types", types).Msg("DNS response found")
		}
	}

	if err := w.WriteMsg(m); err != nil {
		serviceLogger.Warn().Err(err).Msg("Failed to write DNS response")
	}
}

func (s *DNSServer) resolveARequest(q dns.Question, checkCNAME bool) ([]dns.RR, net.IP) {

	if checkCNAME {
		if a, target := s.resolveCNAMERequest(q); a != nil {
			if a, ip := s.resolveARequest(dns.Question{Name: target, Qtype: dns.TypeA}, false); a != nil {
				return a, ip
			}
		}
	}

	if lookup, ok := s.domainIPLoopup[q.Name]; ok {
		return []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  uint16(lookup.domainLookup.class),
					Ttl:    lookup.domainLookup.ttl,
				},
				A: lookup.ip,
			},
		}, lookup.ip
	}

	forwardResponse, err := resolve(q)
	if err == nil {
		return forwardResponse, nil
	}
	return nil, nil
}

func (s *DNSServer) resolveAAAARequest(q dns.Question, checkCNAME bool) ([]dns.RR, net.IP) {
	if checkCNAME {
		if a, target := s.resolveCNAMERequest(q); a != nil {
			if a, ip := s.resolveAAAARequest(dns.Question{Name: target, Qtype: dns.TypeAAAA}, false); a != nil {
				return a, ip
			}
		}
	}

	if lookup, ok := s.domainIPLoopup[q.Name]; ok {
		return []dns.RR{
			&dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeAAAA,
					Class:  uint16(lookup.domainLookup.class),
					Ttl:    lookup.domainLookup.ttl,
				},
				AAAA: lookup.ip,
			},
		}, lookup.ip
	}

	forwardResponse, err := resolve(q)
	if err == nil {
		return forwardResponse, nil
	}

	return nil, nil
}

func (s *DNSServer) resolveCNAMERequest(q dns.Question) ([]dns.RR, string) {
	if lookup, ok := s.domainCNAMELookup[q.Name]; ok {
		return []dns.RR{
			&dns.CNAME{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeCNAME,
					Class:  uint16(lookup.domainLookup.class),
					Ttl:    lookup.domainLookup.ttl,
				},
				Target: lookup.cname,
			},
		}, lookup.cname
	}

	return nil, ""
}

func (s *DNSServer) resolveReverseRequest(q dns.Question) ([]dns.RR, string) {
	if lookup, ok := s.reverseDomainLookup[q.Name]; ok {
		return []dns.RR{
			&dns.PTR{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypePTR,
					Class:  uint16(lookup.domainLookup.class),
					Ttl:    lookup.domainLookup.ttl,
				},
				Ptr: lookup.domain,
			},
		}, lookup.domain
	}

	forwardResponse, err := resolve(q)
	if err == nil {
		return forwardResponse, ""
	}

	return nil, ""
}
