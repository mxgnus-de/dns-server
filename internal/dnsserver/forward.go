package dnsserver

import (
	"github.com/miekg/dns"
)

func resolve(q dns.Question) ([]dns.RR, error) {
	m := &dns.Msg{}
	m.SetQuestion(q.Name, q.Qtype)
	m.RecursionDesired = true

	c := &dns.Client{}
	in, _, err := c.Exchange(m, "1.1.1.1:53")
	if err != nil {
		serviceLogger.Warn().Err(err).Str("name", q.Name).Msg("Failed to resolve DNS request")
		return nil, err
	}

	return in.Answer, nil
}
