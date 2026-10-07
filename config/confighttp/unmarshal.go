// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confighttp // import "go.opentelemetry.io/collector/config/confighttp"

import "go.opentelemetry.io/collector/confmap"

func unmarshalConfig(conf *confmap.Conf, result any) error {
	// The extra squash level lets mapstructure decode the complete config
	// without invoking the legacy Unmarshal hook on its HTTP fields. This
	// preserves unknown-key validation for both HTTP fields and sibling fields.
	view := struct {
		Config any `mapstructure:",squash"`
	}{Config: result}
	return conf.Unmarshal(&view)
}
