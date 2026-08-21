package http

import (
	"fmt"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

// NamedClients is a registry of named, instrumented HTTP clients.
// It provides typed access to pre-configured clients for multi-service scenarios.
// NamedClients is safe for concurrent use from multiple VU goroutines.
type NamedClients struct {
	clients map[string]*Client
}

// Get returns the named client and a boolean indicating if it exists.
func (nc *NamedClients) Get(name string) (*Client, bool) {
	c, ok := nc.clients[name]
	return c, ok
}

// MustGet returns the named client or panics if it doesn't exist.
// Use this when the client name is guaranteed to be valid (configured in vuhive.yaml).
func (nc *NamedClients) MustGet(name string) *Client {
	c, ok := nc.clients[name]
	if !ok {
		panic(fmt.Sprintf("vuhive/http: named client %q not found in registry", name))
	}
	return c
}

// Names returns the names of all registered clients.
func (nc *NamedClients) Names() []string {
	names := make([]string, 0, len(nc.clients))
	for name := range nc.clients {
		names = append(names, name)
	}
	return names
}

// Len returns the number of registered clients.
func (nc *NamedClients) Len() int {
	return len(nc.clients)
}

// NamedClientsProvider provides both metrics and named HTTP client configs from a context.
type NamedClientsProvider interface {
	HTTPClients() map[string]vuhive.HTTPConfig
	Metrics() vuhive.MetricsCollector
}

// NewNamedClientsFromConfig creates a NamedClients registry from a context's
// declarative http_clients configuration. Each named client is constructed
// with its own base URL, headers, timeout, TLS, pool, and metric prefix.
func NewNamedClientsFromConfig(ctx NamedClientsProvider, overrides ...Option) *NamedClients {
	configs := ctx.HTTPClients()
	clients := make(map[string]*Client, len(configs))
	for name, cfg := range configs {
		clients[name] = newClientFromHTTPConfig(cfg, ctx.Metrics(), overrides...)
	}
	return &NamedClients{clients: clients}
}
