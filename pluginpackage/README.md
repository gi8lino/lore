# Lore plugin packages

This public package owns the single versioned manifest and bounded ZIP validation
contract used by Lore and the Go developer CLI. `ParseManifest` strictly decodes
one manifest; `Read` validates a complete `.loreplugin`. Returned packages expose
defensive copies and archives are never extracted by the runtime.

See [the SDK](../pluginsdk/README.md) for project creation and deterministic builds,
and [the runtime architecture](../internal/plugin/README.md) for lifecycle and
security boundaries.
