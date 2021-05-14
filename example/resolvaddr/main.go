package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
	flag "github.com/spf13/pflag"
)

type CliOpt struct {
	Nameserver string
	Port       uint
	Host       string
	SrcAddr    string
}

func main() {

	opt := &CliOpt{}
	flag.StringVar(&opt.Nameserver, "ns", "", "Name server address (use system DNS as default)")
	flag.UintVarP(&opt.Port, "port", "p", 53, "Name server UDP port")
	flag.StringVar(&opt.Host, "host", "google.com", "Name of the host to resolve (search for A records)")
	flag.StringVarP(&opt.SrcAddr, "from-address", "I", "", "Local IP address to send request from")
	flag.Parse()

	var dnsAddr string
	if len(opt.Nameserver) == 0 {
		resolvConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			log.Fatalf("parse /etc/resolv.conf: %v", err)
		}

		dnsAddr = fmt.Sprintf("%s:%s", resolvConfig.Servers[0], resolvConfig.Port)
	} else {
		dnsAddr = fmt.Sprintf("%s:%d", opt.Nameserver, opt.Port)
	}
	log.Printf("Using dns server %s", dnsAddr)

	c := new(dns.Client)

	c.Dialer = &net.Dialer{
		Timeout: 500 * time.Millisecond,
	}
	if len(opt.SrcAddr) > 0 {
		c.Dialer.LocalAddr = &net.UDPAddr{
			IP:   net.ParseIP("1.2.3.4"),
			Port: 0,
			Zone: "",
		}
	}

	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{dns.Fqdn(opt.Host), dns.TypeA, dns.ClassINET}

	expireCtx, notExpired := context.WithTimeout(context.Background(), 3*time.Second)
	defer notExpired()
	in, rtt, err := c.ExchangeContext(expireCtx, m1, dnsAddr)
	if err != nil {
		log.Printf("DNS request error after %v: %v", rtt, err)
		return
	}
	if in.Rcode != dns.RcodeSuccess {
		log.Fatalf("Unexpected RCode: %d %v", in.Rcode, dns.RcodeToString[in.Rcode])
	}

	log.Printf("Response in %v", rtt)
	for i, rr := range in.Answer {
		switch rrA := rr.(type) {
		case *dns.A:
			log.Printf("RR #%d: TTL: %v; IPADDR: %s", i, time.Duration(rr.Header().Ttl)*time.Second, rrA.A)
		case *dns.AAAA:
			log.Printf("RR #%d: TTL: %v; IPADDR: %s", i, time.Duration(rr.Header().Ttl)*time.Second, rrA.AAAA)
		default:
			log.Printf("RR #%d: UNEXPECTED TYPE: %s ", i, rr.String())
		}

	}
}
