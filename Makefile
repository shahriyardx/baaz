BIN     := $(HOME)/.local/bin/baaz
PLUGIN  := $(HOME)/.config/omarchy/plugins/shahriyardx.baaz
APP     := $(HOME)/Applications/Baaz.app
BUILD   := build
VERSION := 0.0.0-dev

.PHONY: build test install install-plugin install-chrome uninstall \
        extension-store extension-screenshots \
        macos-app macos-dmg macos-test macos-install macos-check

build:
	go build -o baaz ./cmd/baaz

# A Chrome Web Store upload: the same extension with the "key" field stripped,
# because the store assigns its own ID and rejects a package carrying one.
extension-store:
	./extension/pack-store.sh

# 1280x800 listing screenshots, rendered from the real popup markup.
extension-screenshots:
	./extension/store-assets/render.sh $(BUILD)/store-screenshots

test:
	go test ./...

install: build
	mkdir -p $(dir $(BIN))
	rm -f $(BIN)
	install -m 755 baaz $(BIN)
	@echo "installed $(BIN)"

install-plugin:
	mkdir -p $(PLUGIN)
	install -m 644 bar-plugin/manifest.json bar-plugin/Panel.qml $(PLUGIN)/
	omarchy-shell shell rescanPlugins || true
	omarchy plugin enable shahriyardx.baaz || true
	omarchy bar move shahriyardx.baaz --section right || true
	@echo "bar plugin installed"

# usage: make install-chrome EXT_ID=<id from chrome://extensions>
install-chrome: install
	$(BIN) install-chrome --ext-id $(EXT_ID)

uninstall:
	-pkill -f 'baaz daemon'
	-pkill -f 'BaazMenuBar'
	-launchctl bootout gui/$(shell id -u)/com.shahriyar.baaz.menubar 2>/dev/null
	rm -f $(BIN)
	rm -rf $(PLUGIN)
	rm -rf $(APP)
	rm -f $(HOME)/Library/LaunchAgents/com.shahriyar.baaz.menubar.plist
	rm -f $(HOME)/.config/google-chrome/NativeMessagingHosts/com.shahriyar.baaz.json
	rm -f $(HOME)/.config/chromium/NativeMessagingHosts/com.shahriyar.baaz.json
	rm -f "$(HOME)/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.shahriyar.baaz.json"
	rm -f "$(HOME)/Library/Application Support/Chromium/NativeMessagingHosts/com.shahriyar.baaz.json"
	rm -rf "$(HOME)/Downloads/baaz-extension"

# ---------- macOS ----------

# Baaz.app is the SwiftUI menu bar widget — the macOS counterpart to the
# Omarchy bar plugin.
macos-app:
	./macos/make-app.sh $(BUILD)

# The shippable macOS artifact: app + embedded CLI, as one drag.
macos-dmg:
	./macos/make-dmg.sh $(BUILD) $(VERSION)

# Swift 6 language mode is stricter than the default, and stricter than some
# CI toolchains — building under it locally is what stops a concurrency error
# reaching the release runner.
macos-strict:
	swift build --package-path macos/menubar -c release -Xswiftc -swift-version -Xswiftc 6

macos-test: macos-strict
	swift test --package-path macos/menubar

# XProtect deletes Go binaries that match its adware signature; see the
# script. Run this on anything shipped to users.
macos-check: build
	./macos/xprotect-check.sh baaz

macos-install: macos-check macos-app
	mkdir -p $(dir $(BIN))
	rm -f $(BIN)
	install -m 755 baaz $(BIN)
	@echo "installed $(BIN)"
	$(BIN) install-menubar
