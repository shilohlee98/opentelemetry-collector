// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zpagesextension

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestUnmarshalDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.New().Unmarshal(&cfg))
	assert.Equal(t, factory.CreateDefaultConfig(), cfg)
}

func TestUnmarshalConfigUnknownKeys(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input map[string]any
		key   string
	}{
		{"tls typo", map[string]any{"tls_settings": map[string]any{"cert_file": "server.crt"}}, "tls_settings"},
		{"expvar typo", map[string]any{"expavr": map[string]any{"enabled": true}}, "expavr"},
		{"nested expvar typo", map[string]any{"expvar": map[string]any{"enable": true}}, "enable"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewFactory().CreateDefaultConfig()
			cm := confmap.NewFromStringMap(tt.input)
			assert.ErrorContains(t, cm.Unmarshal(cfg), "invalid keys: "+tt.key)
		})
	}
}

func TestInvalidConfig(t *testing.T) {
	assert.Error(t, (&Config{}).Validate())
}

func TestUnmarshalConfig(t *testing.T) {
	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	expectedServerConfig := confighttp.NewDefaultServerConfig()
	expectedServerConfig.NetAddr.Endpoint = "localhost:56888"

	assert.Equal(t, &Config{ServerConfig: expectedServerConfig}, cfg)
}

// Fields declared next to the squashed ServerConfig must still be decoded.
func TestUnmarshalConfigWithExpvar(t *testing.T) {
	cm := confmap.NewFromStringMap(map[string]any{
		"endpoint": "localhost:56888",
		"expvar":   map[string]any{"enabled": true},
	})
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, cm.Unmarshal(&cfg))

	zCfg := cfg.(*Config)
	assert.Equal(t, "localhost:56888", zCfg.ServerConfig.NetAddr.Endpoint)
	assert.True(t, zCfg.Expvar.Enabled)
}
