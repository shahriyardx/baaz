BIN     := $(HOME)/.local/bin/dm
PLUGIN  := $(HOME)/.config/omarchy/plugins/shahriyar.downloads

.PHONY: build test install install-plugin install-chrome uninstall

build:
	go build -o dm ./cmd/dm

test:
	go test ./...

install: build
	mkdir -p $(dir $(BIN))
	install -m 755 dm $(BIN)
	@echo "installed $(BIN)"

install-plugin:
	mkdir -p $(PLUGIN)
	install -m 644 bar-plugin/manifest.json bar-plugin/Panel.qml $(PLUGIN)/
	omarchy-shell shell rescanPlugins || true
	omarchy plugin enable shahriyar.downloads || true
	omarchy bar move shahriyar.downloads --section right || true
	@echo "bar plugin installed"

# usage: make install-chrome EXT_ID=<id from chrome://extensions>
install-chrome: install
	$(BIN) install-chrome --ext-id $(EXT_ID)

uninstall:
	-pkill -f 'dm daemon'
	rm -f $(BIN)
	rm -rf $(PLUGIN)
	rm -f $(HOME)/.config/google-chrome/NativeMessagingHosts/com.shahriyar.dm.json
	rm -f $(HOME)/.config/chromium/NativeMessagingHosts/com.shahriyar.dm.json
