package tfplugin

import (
	"github.com/hashicorp/go-plugin"
)

// terraform@v.1.4.5/internal/plugin/plugin.go

// VersionedPlugins includes both protocol 5 and 6 because this is the function
// called in providerFactory (command/meta_providers.go) to set up the initial
// plugin client config.
var VersionedPlugins = map[int]plugin.PluginSet{
	5: {
		ProviderPluginName: &GRPCProviderPlugin{},
	},
}

// terraform@v.1.4.5/internal/plugin/serve.go

const (
	// The constants below are the names of the plugins that can be dispensed
	// from the plugin server.
	ProviderPluginName = "provider"

	// DefaultProtocolVersion is the protocol version assumed for legacy clients that don't specify
	// a particular version during their handshake. This is the version used when Terraform 0.10
	// and 0.11 launch plugins that were built with support for both versions 4 and 5, and must
	// stay unchanged at 4 until we intentionally build plugins that are not compatible with 0.10 and
	// 0.11.
	DefaultProtocolVersion = 4
)

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	// The ProtocolVersion is the version that must match between TF core
	// and TF plugins. This should be bumped whenever a change happens in
	// one or the other that makes it so that they can't safely communicate.
	// This could be adding a new interface value, it could be how
	// helper/schema computes diffs, etc.
	ProtocolVersion: DefaultProtocolVersion,

	// The magic cookie values should NEVER be changed.
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}
