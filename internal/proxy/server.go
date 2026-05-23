package proxy

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/miekg/dns"
	"github.com/whooslizi/dns-fetching/internal/cache"
	"github.com/whooslizi/dns-fetching/internal/doh"
)

type Server struct {
	udpServer  *dns.Server
	tcpServer  *dns.Server
	dohClient  *doh.Client
	cache      *cache.Cache
	listenAddr string
	running    atomic.Bool
	mu         sync.RWMutex
	stats      Stats
}

type Stats struct {
	QueriesServed uint64
}

func New(listenAddr string, dohClient *doh.Client, dnsCache *cache.Cache) *Server {
	return &Server{
		dohClient:  dohClient,
		cache:      dnsCache,
		listenAddr: listenAddr,
	}
}

func (s *Server) Start() error {
	handler := dns.HandlerFunc(s.handleQuery)

	s.udpServer = &dns.Server{Addr: s.listenAddr, Net: "udp", Handler: handler}
	s.tcpServer = &dns.Server{Addr: s.listenAddr, Net: "tcp", Handler: handler}

	errCh := make(chan error, 2)
	go func() { errCh <- s.udpServer.ListenAndServe() }()
	go func() { errCh <- s.tcpServer.ListenAndServe() }()

	select {
	case err := <-errCh:
		return fmt.Errorf("dns server failed to start: %w", err)
	default:
		s.running.Store(true)
		return nil
	}
}

func (s *Server) Stop() error {
	s.running.Store(false)
	var firstErr error
	if s.udpServer != nil {
		if err := s.udpServer.Shutdown(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.tcpServer != nil {
		if err := s.tcpServer.Shutdown(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *Server) IsRunning() bool {
	return s.running.Load()
}

func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		s.refuse(w, r)
		return
	}

	q := r.Question[0]

	msg, err := s.cache.GetOrFetch(q.Name, q.Qtype, q.Qclass, func() (*dns.Msg, error) {
		return s.dohClient.Exchange(context.Background(), r)
	})

	if err != nil {
		log.Printf("upstream failed for %s: %v", q.Name, err)
		s.servfail(w, r)
		return
	}

	msg.SetReply(r)
	msg.Answer = msg.Answer
	msg.Ns = msg.Ns
	msg.Extra = msg.Extra
	msg.Rcode = msg.Rcode

	s.mu.Lock()
	s.stats.QueriesServed++
	s.mu.Unlock()

	w.WriteMsg(msg)
}

func (s *Server) servfail(w dns.ResponseWriter, r *dns.Msg) {
	resp := new(dns.Msg)
	resp.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(resp)
}

func (s *Server) refuse(w dns.ResponseWriter, r *dns.Msg) {
	resp := new(dns.Msg)
	resp.SetRcode(r, dns.RcodeRefused)
	w.WriteMsg(resp)
}

func (s *Server) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}
