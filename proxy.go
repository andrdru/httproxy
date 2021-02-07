package main

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type (
	server struct {
		host      string
		endpoint  string
		proxy     *httputil.ReverseProxy
		isHealthy bool
		client    *http.Client
		mu        sync.Mutex
	}
)

var (
	servers          []*server
	serverIndex      int
	serverIndexMutex = sync.Mutex{}
)

func newServer(host string, endpoint string, requestTimeout time.Duration, client *http.Client) *server {
	var u = &url.URL{
		Scheme: "http",
		Host:   host,
	}

	var endpointUrl = u

	endpointUrl.Path = endpoint

	var s = &server{
		host:     host,
		endpoint: endpointUrl.String(),
		client:   client,
		proxy:    httputil.NewSingleHostReverseProxy(u),
		mu:       sync.Mutex{},
	}

	s.proxy.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   requestTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return s
}

func (s *server) healthCheck(t *time.Ticker) {
	<-t.C

	resp, err := s.client.Get(s.endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("server %s unhealthy", s.host)
		s.mu.Lock()
		s.isHealthy = false
		s.mu.Unlock()

		return
	}

	if !s.isHealthy {
		log.Printf("server %s healthy", s.host)
		s.mu.Lock()
		s.isHealthy = true
		s.mu.Unlock()
	}
}
