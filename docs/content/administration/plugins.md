# Plugins

Open **Administration → Plugins** to manage Lore's bundled and installed plugins.
Only administrators can view these pages or change plugin lifecycle state.

The list shows each plugin's name, version, provider, source, and status and lets administrators enable or disable optional plugins directly. Select a plugin to open its detail modal without leaving the plugin inventory. The modal contains its packaged documentation, plugin-owned settings, lifecycle controls, upgrade and uninstall actions, plus its ID, modules, requested permissions, dependencies, and technical metadata. Provider names are supplied by package authors; they are not verification badges.

## Install and upgrade

Upload one `.loreplugin` package, up to 16 MiB, and select **Install and enable**.
Lore validates its format, API compatibility, dependencies, and permission policy
before publishing its contributions. Installation takes effect immediately.
A package requesting capabilities that Lore does not grant cannot be installed.
There is no marketplace or automatic package download.

To upgrade, open the plugin and upload a package with the same plugin ID.
Lore validates the replacement before switching versions. Enabled plugins stay
enabled, disabled plugins stay disabled, and a failed upgrade preserves the
current version. Upgrading a bundled plugin creates an installed override through
the same package loader and runtime.

## Enable, disable, and uninstall

**Disable plugin** removes its contributions from new renders. Active renders
finish using their existing version; browser modules notice the change shortly
afterward and restore their HTML/source fallback. **Enable plugin** restores its
contributions without restarting Lore.

Dependencies must remain enabled while a dependent plugin is active. System
plugins marked required by deployment policy cannot be disabled or removed.
Packages cannot declare themselves required.

Installed plugins can be uninstalled from their detail modal. Plugin settings and
namespaced data are retained for reinstall. Removing an installed override leaves
any bundled copy disabled. Bundled plugins themselves can be disabled; their
embedded package remains part of Lore.

Plugin-owned rendering features and their settings live in the plugin detail modal. **Administration → Rendering** contains only Lore's built-in Markdown behavior. Declarative plugin settings are boolean controls; arbitrary custom settings editors are not provided yet.
