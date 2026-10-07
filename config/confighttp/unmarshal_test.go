// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp

import (
	"maps"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/confighttp/internal/metadata"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
)

type strictClientConfig namedSquashClientConfig

func (cfg *strictClientConfig) Unmarshal(conf *confmap.Conf) error {
	return cfg.ClientConfig.UnmarshalConfig(conf, (*namedSquashClientConfig)(cfg))
}

type strictServerConfig namedSquashServerConfig

func (cfg *strictServerConfig) Unmarshal(conf *confmap.Conf) error {
	return cfg.ServerConfig.UnmarshalConfig(conf, (*namedSquashServerConfig)(cfg))
}

func TestUnmarshalConfig(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run("prioritize_new_keepalive="+strconv.FormatBool(enabled), func(t *testing.T) {
			gate := metadata.PkgConfighttpPrioritizeNewKeepaliveFeatureGate
			previous := gate.IsEnabled()
			require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), enabled))
			t.Cleanup(func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(gate.ID(), previous))
			})

			for _, tt := range []struct {
				name      string
				client    map[string]any
				server    map[string]any
				wantError string
			}{
				{name: "programmatic defaults"},
				{
					name:   "deprecated fields",
					client: map[string]any{"idle_conn_timeout": "45s"},
					server: map[string]any{"idle_timeout": "45s"},
				},
				{
					name:   "keepalive section",
					client: map[string]any{"keepalive": map[string]any{"idle_conn_timeout": "45s"}},
					server: map[string]any{"keepalive": map[string]any{"idle_timeout": "45s"}},
				},
				{
					name:   "disabled keepalive",
					client: map[string]any{"keepalive": map[string]any{"enabled": false}},
					server: map[string]any{"keepalive": map[string]any{"enabled": false}},
				},
				{
					name:   "null keepalive",
					client: map[string]any{"keepalive": nil},
					server: map[string]any{"keepalive": nil},
				},
				{
					name:      "conflicting keepalive fields",
					client:    map[string]any{"idle_conn_timeout": "1m", "keepalive": map[string]any{}},
					server:    map[string]any{"idle_timeout": "1m", "keepalive": map[string]any{}},
					wantError: "cannot use deprecated keepalive fields",
				},
				{
					name:      "unknown field",
					client:    map[string]any{"tls_settings": map[string]any{"cert_file": "client.crt"}},
					server:    map[string]any{"tls_settings": map[string]any{"cert_file": "server.crt"}},
					wantError: "invalid keys: tls_settings",
				},
				{
					name:      "unknown keepalive field",
					client:    map[string]any{"keepalive": map[string]any{"typo": true}},
					server:    map[string]any{"keepalive": map[string]any{"typo": true}},
					wantError: "invalid keys: typo",
				},
				{
					name:      "invalid field type",
					client:    map[string]any{"timeout": "invalid"},
					server:    map[string]any{"read_timeout": "invalid"},
					wantError: "invalid duration",
				},
			} {
				t.Run(tt.name, func(t *testing.T) {
					clientInput := map[string]any{"endpoint": "http://localhost:4318", "extra": "sibling"}
					serverInput := map[string]any{"endpoint": "localhost:4318", "extra": "sibling"}
					maps.Copy(clientInput, tt.client)
					maps.Copy(serverInput, tt.server)
					client := strictClientConfig{ClientConfig: NewDefaultClientConfig()}
					server := strictServerConfig{ServerConfig: NewDefaultServerConfig()}
					// Exercise programmatic defaults as well as values from the input.
					client.ClientConfig.Keepalive = configoptional.Some(KeepaliveClientConfig{
						IdleConnTimeout: 3 * time.Minute,
						MaxIdleConns:    10,
					})
					server.ServerConfig.Keepalive = configoptional.Some(KeepaliveServerConfig{
						IdleTimeout: 3 * time.Minute,
					})
					legacyClient := namedSquashClientConfig{ClientConfig: client.ClientConfig}
					legacyServer := namedSquashServerConfig{ServerConfig: server.ServerConfig}
					clientConf := confmap.NewFromStringMap(clientInput)
					serverConf := confmap.NewFromStringMap(serverInput)
					clientErr := clientConf.Unmarshal(&client)
					serverErr := serverConf.Unmarshal(&server)
					if tt.wantError != "" {
						require.ErrorContains(t, clientErr, tt.wantError)
						require.ErrorContains(t, serverErr, tt.wantError)
						if tt.name == "conflicting keepalive fields" {
							require.ErrorContains(t, clientConf.Unmarshal(&legacyClient), tt.wantError)
							require.ErrorContains(t, serverConf.Unmarshal(&legacyServer), tt.wantError)
						}
						return
					}
					require.NoError(t, clientErr)
					require.NoError(t, serverErr)
					assert.Equal(t, "sibling", client.Extra)
					assert.Equal(t, "sibling", server.Extra)
					// The strict path must preserve migration, defaults, and warnings
					// for downstream components still using the legacy hook.
					require.NoError(t, clientConf.Unmarshal(&legacyClient))
					require.NoError(t, serverConf.Unmarshal(&legacyServer))
					assert.Equal(t, legacyClient, namedSquashClientConfig(client))
					assert.Equal(t, legacyServer, namedSquashServerConfig(server))
				})
			}
		})
	}
}
