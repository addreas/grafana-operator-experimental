package v1beta1

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/blang/semver"
)

type GrafanaPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PluginList []GrafanaPlugin

type PluginMap map[string]PluginList

func (l PluginList) Hash() string {
	sb := strings.Builder{}
	for _, plugin := range l {
		sb.WriteString(plugin.Name)
		sb.WriteString(plugin.Version)
	}
	hash := sha256.New()
	io.WriteString(hash, sb.String()) // nolint
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (l PluginList) String() string {
	var plugins []string
	for _, plugin := range l {
		plugins = append(plugins, fmt.Sprintf("%s %s", plugin.Name, plugin.Version))
	}
	return strings.Join(plugins, ",")
}

// Update update plugin version
func (l PluginList) Update(plugin *GrafanaPlugin) {
	for _, installedPlugin := range l {
		if installedPlugin.Name == plugin.Name {
			installedPlugin.Version = plugin.Version
			break
		}
	}
}

// Sanitize remove duplicates and enforce semver
func (l PluginList) Sanitize() PluginList {
	var sanitized PluginList
	for _, plugin := range l {
		_, err := semver.Parse(plugin.Version)
		if err != nil {
			continue
		}
		if !sanitized.HasSomeVersionOf(&plugin) {
			sanitized = append(sanitized, plugin)
		}
	}
	return sanitized
}

// HasSomeVersionOf returns true if the list contains the same plugin in the exact or a different version
func (l PluginList) HasSomeVersionOf(plugin *GrafanaPlugin) bool {
	for _, listedPlugin := range l {
		if listedPlugin.Name == plugin.Name {
			return true
		}
	}
	return false
}

// GetInstalledVersionOf gets the plugin from the list regardless of the version
func (l PluginList) GetInstalledVersionOf(plugin *GrafanaPlugin) *GrafanaPlugin {
	for _, listedPlugin := range l {
		if listedPlugin.Name == plugin.Name {
			return &listedPlugin
		}
	}
	return nil
}

// HasExactVersionOf returns true if the list contains the same plugin in the same version
func (l PluginList) HasExactVersionOf(plugin *GrafanaPlugin) bool {
	for _, listedPlugin := range l {
		if listedPlugin.Name == plugin.Name && listedPlugin.Version == plugin.Version {
			return true
		}
	}
	return false
}

// HasNewerVersionOf returns true if the list contains the same plugin but in a newer version
func (l PluginList) HasNewerVersionOf(plugin *GrafanaPlugin) (bool, error) {
	for _, listedPlugin := range l {
		if listedPlugin.Name != plugin.Name {
			continue
		}

		listedVersion, err := semver.Make(listedPlugin.Version)
		if err != nil {
			return false, err
		}

		requestedVersion, err := semver.Make(plugin.Version)
		if err != nil {
			return false, err
		}

		if listedVersion.Compare(requestedVersion) == 1 {
			return true, nil
		}
	}
	return false, nil
}

// VersionsOf returns the number of different versions of a given plugin in the list
func (l PluginList) VersionsOf(plugin *GrafanaPlugin) int {
	i := 0
	for _, listedPlugin := range l {
		if listedPlugin.Name == plugin.Name {
			i = i + 1
		}
	}
	return i
}

func (l PluginList) ConsolidatedConcat(others PluginList) (PluginList, error) {
	var consolidatedPlugins PluginList

	for _, plugin := range others {
		// new plugin
		if !consolidatedPlugins.HasSomeVersionOf(&plugin) {
			consolidatedPlugins = append(consolidatedPlugins, plugin)
			continue
		}

		// newer version of plugin already installed
		hasNewer, err := consolidatedPlugins.HasNewerVersionOf(&plugin)
		if err != nil {
			return nil, err
		}

		if hasNewer {
			continue
		}

		// duplicate plugin
		if consolidatedPlugins.HasExactVersionOf(&plugin) {
			continue
		}

		// some version is installed, but it is not newer and it is not the same: must be older
		consolidatedPlugins.Update(&plugin)
	}
	return consolidatedPlugins, nil
}
