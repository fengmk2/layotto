package wasm

import (
	"os"
	"path/filepath"
	"strings"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/wasm"

	"github.com/fsnotify/fsnotify"
)

var (
	watcher *fsnotify.Watcher
	// map[wasm-file-path]config
	configs = make(map[string]*filterConfigItem)
	// map[wasm-file-path]Factory
	factories = make(map[string]*FilterConfigFactory)
)

func init() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.DefaultLogger.Errorf("[proxywasm] [watcher] init fail to create watcher: %v", err)
		return
	}
	go runWatcher()
}

func runWatcher() {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.DefaultLogger.Errorf("[proxywasm] [watcher] runWatcher exit")
				return
			}
			log.DefaultLogger.Debugf("[proxywasm] [watcher] runWatcher got event, %s", event)

			if pathIsWasmFile(event.Name) {
				if event.Op&fsnotify.Chmod == fsnotify.Chmod ||
					event.Op&fsnotify.Rename == fsnotify.Rename {
					continue
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					// rewatch the file if it exists
					// remove this file then nename other file to this name will cause this case
					if fileExist(event.Name) {
						_ = watcher.Add(event.Name)
					}
					continue
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					if fileExist(event.Name) {
						_ = watcher.Add(event.Name)
					}
				}
				reloadWasm(event.Name)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				log.DefaultLogger.Errorf("[proxywasm] [watcher] runWatcher exit")
				return
			}
			log.DefaultLogger.Errorf("[proxywasm] [watcher] runWatcher got errors, err: %v", err)
		}
	}
}

func addWatchFile(cfg *filterConfigItem, factory *FilterConfigFactory) {
	path := cfg.VmConfig.Path
	if err := watcher.Add(path); err != nil {
		log.DefaultLogger.Errorf("[proxywasm] [watcher] addWatchFile fail to watch wasm file, err: %v", err)
	}

	dir := filepath.Dir(path)
	if err := watcher.Add(dir); err != nil {
		log.DefaultLogger.Errorf("[proxywasm] [watcher] addWatchFile fail to watch wasm dir, err: %v", err)
		return
	}

	configs[path] = cfg
	factories[path] = factory
	log.DefaultLogger.Infof("[proxywasm] [watcher] addWatchFile start to watch wasm file and its dir: %s", path)
}

func reloadWasm(fullPath string) {
	found := false

	for path, config := range configs {
		if strings.HasSuffix(fullPath, path) {
			found = true

			vmConfig := *config.VmConfig
			vmConfig.Md5 = ""
			v2Config := v2.WasmPluginConfig{
				PluginName:  config.PluginName,
				VmConfig:    &vmConfig,
				InstanceNum: config.InstanceNum,
			}
			err := wasm.GetWasmManager().AddOrUpdateWasm(v2Config)
			if err != nil {
				log.DefaultLogger.Errorf("[proxywasm] [watcher] reloadWasm fail to add plugin, err: %v", err)
				return
			}

			pw := wasm.GetWasmManager().GetWasmPluginWrapperByName(config.PluginName)
			if pw == nil {
				log.DefaultLogger.Errorf("[proxywasm] [watcher] reloadWasm plugin not found")
				return
			}

			factory := factories[path]
			config.VmConfig = pw.GetConfig().VmConfig
			factory.config = append(factory.config, config)

			wasmPlugin := &WasmPlugin{
				pluginName:    config.PluginName,
				plugin:        pw.GetPlugin(),
				rootContextID: config.RootContextID,
				config:        config,
			}
			factory.plugins[config.PluginName] = wasmPlugin

			pw.RegisterPluginHandler(factory)

			for _, plugin := range factory.plugins {
				if plugin.pluginName == config.PluginName {
					plugin.plugin = pw.GetPlugin()
				}
			}
			log.DefaultLogger.Infof("[proxywasm] [watcher] reloadWasm reload wasm success: %s", path)
		}
	}

	if !found {
		log.DefaultLogger.Errorf("[proxywasm] [watcher] reloadWasm WasmPluginConfig not found: %s", fullPath)
	}
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

func pathIsWasmFile(fullPath string) bool {
	for path, _ := range configs {
		if strings.HasSuffix(fullPath, path) {
			return true
		}
	}
	return false
}
