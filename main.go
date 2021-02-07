package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type (
	balance string

	flags struct {
		address             string
		hosts               string
		endpoint            string
		timeout             int64
		healthCheckInterval int64
		healthCheckTimeout  int64
		balance             string
	}

	config struct {
		hosts               []string
		endpoint            string
		healthCheckInterval time.Duration
		healthCheckTimeout  time.Duration
		timeout             time.Duration
		balance             balance
	}
)

const (
	balanceRoundRobin balance = "round_robin"
	balanceDisable    balance = "disable"
)

var (
	balances = map[balance]bool{
		balanceRoundRobin: true,
		balanceDisable:    true,
	}
)

func main() {
	var f = parseFlags()

	var conf = initConfig(f)

	if err := conf.validate(); err != nil {
		log.Fatal(err)
	}

	var httpClient = &http.Client{Timeout: conf.healthCheckTimeout}

	for _, host := range conf.hosts {
		s := newServer(host, conf.endpoint, conf.timeout, httpClient)
		s.proxy.ErrorHandler = ProxyErrorHandler

		go func(s *server, interval time.Duration) {
			var t = time.NewTicker(interval)
			for {
				s.healthCheck(t)
				t.Reset(interval)
			}
		}(s, conf.healthCheckInterval)

		servers = append(servers, s)
	}

	http.HandleFunc("/", BalanceHandler)
	log.Fatal(http.ListenAndServe(f.address, nil))

}

func BalanceHandler(writer http.ResponseWriter, request *http.Request) {
	if len(servers) == 0 {
		return
	}

	var current = chooseServer()
	if current == nil {
		writer.WriteHeader(http.StatusBadGateway)
		writer.Write([]byte("empty proxy pool"))
		return
	}

	current.proxy.ServeHTTP(writer, request)
}

func ProxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	BalanceHandler(w, r)
}

func chooseServer() (current *server) {
	var count = 0

	for {
		if count > len(servers) {
			break
		}

		if serverIndex > len(servers)-1 {
			serverIndex = 0
		}
		current = servers[serverIndex]

		serverIndexMutex.Lock()
		serverIndex++
		serverIndexMutex.Unlock()

		if current.isHealthy {
			return current
		}

		count++
	}

	return nil
}

func parseFlags() flags {
	var f flags

	flag.StringVar(&f.address, "address", ":8080", "balancer address")
	flag.StringVar(&f.hosts, "hosts", "", "list of hosts or IPs, delimited with ;")

	flag.StringVar(
		&f.endpoint,
		"endpoint",
		"/health",
		"health check endpoint on hosts, provide HTTP 200 OK if healthy",
	)

	flag.Int64Var(&f.healthCheckInterval, "interval", 1000, "time in ms, repeat interval")
	flag.Int64Var(&f.healthCheckTimeout, "health_timeout", 500, "time in ms, health check timeout")
	flag.Int64Var(&f.timeout, "timeout", 5000, "time in ms, health check timeout")

	flag.StringVar(
		&f.balance,
		"balance",
		balanceRoundRobin.String(),
		"proxy balancing, one of: round_robin, disable",
	)

	var isHelp bool
	flag.BoolVar(&isHelp, "help", false, "print help")

	flag.Parse()

	if isHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	return f
}

func (s balance) String() string {
	return string(s)
}

func initConfig(f flags) config {
	var c = config{
		endpoint:            f.endpoint,
		balance:             balance(f.balance),
		healthCheckTimeout:  time.Duration(f.healthCheckTimeout) * time.Millisecond,
		healthCheckInterval: time.Duration(f.healthCheckInterval) * time.Millisecond,
		timeout:             time.Duration(f.timeout) * time.Millisecond,
	}

	c.hosts = strings.Split(strings.Trim(f.hosts, "; "), ";")
	for i := range c.hosts {
		c.hosts[i] = strings.Trim(c.hosts[i], " ")
	}

	return c
}

func (c *config) validate() error {
	if len(c.hosts) == 0 {
		return errors.New("empty hosts list")
	}

	if !balances[c.balance] {
		return errors.New("wrong balance type: " + c.balance.String())
	}

	if c.healthCheckInterval == 0 {
		return errors.New("health check interval could not be 0")
	}

	if c.healthCheckTimeout == 0 {
		return errors.New("health check timeout could not be 0")
	}

	if c.timeout == 0 {
		return errors.New("proxy timeout could not be 0")
	}

	return nil
}
