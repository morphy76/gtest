package http

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockNamedClientsProvider struct {
	configs map[string]vuhive.HTTPConfig
	metrics vuhive.MetricsCollector
}

func (m *mockNamedClientsProvider) HTTPClients() map[string]vuhive.HTTPConfig { return m.configs }
func (m *mockNamedClientsProvider) Metrics() vuhive.MetricsCollector          { return m.metrics }

func TestNamedClients_NewNamedClientsFromConfig(t *testing.T) {
	provider := &mockNamedClientsProvider{
		configs: map[string]vuhive.HTTPConfig{
			"auth": {
				BaseURL: "https://auth.example.com",
				Timeout: 2 * time.Second,
			},
			"api": {
				BaseURL: "https://api.example.com",
				Timeout: 5 * time.Second,
			},
		},
		metrics: nil,
	}

	nc := NewNamedClientsFromConfig(provider)
	require.NotNil(t, nc)
	assert.Equal(t, 2, nc.Len())

	names := nc.Names()
	assert.ElementsMatch(t, []string{"auth", "api"}, names)
}

func TestNamedClients_Get_ReturnsClientAndTrue(t *testing.T) {
	provider := &mockNamedClientsProvider{
		configs: map[string]vuhive.HTTPConfig{
			"auth": {BaseURL: "https://auth.example.com"},
		},
	}
	nc := NewNamedClientsFromConfig(provider)

	client, ok := nc.Get("auth")
	assert.True(t, ok)
	assert.NotNil(t, client)
	assert.Equal(t, "https://auth.example.com", client.BaseURL())
}

func TestNamedClients_Get_MissingReturnsFalse(t *testing.T) {
	provider := &mockNamedClientsProvider{
		configs: map[string]vuhive.HTTPConfig{},
	}
	nc := NewNamedClientsFromConfig(provider)

	client, ok := nc.Get("missing")
	assert.False(t, ok)
	assert.Nil(t, client)
}

func TestNamedClients_MustGet_Returns(t *testing.T) {
	provider := &mockNamedClientsProvider{
		configs: map[string]vuhive.HTTPConfig{
			"api": {BaseURL: "https://api.example.com"},
		},
	}
	nc := NewNamedClientsFromConfig(provider)

	client := nc.MustGet("api")
	assert.NotNil(t, client)
	assert.Equal(t, "https://api.example.com", client.BaseURL())
}

func TestNamedClients_MustGet_Panics(t *testing.T) {
	provider := &mockNamedClientsProvider{
		configs: map[string]vuhive.HTTPConfig{},
	}
	nc := NewNamedClientsFromConfig(provider)

	assert.PanicsWithValue(t, `vuhive/http: named client "missing" not found in registry`, func() {
		nc.MustGet("missing")
	})
}
