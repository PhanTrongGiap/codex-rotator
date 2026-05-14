.PHONY: fmt test build lint clean install uninstall

fmt:
	go fmt ./...

lint:
	gofmt -l ./cmd ./internal

test:
	go test -race -cover ./...

build:
	go build -o codex-rotator ./cmd/codex-rotator

clean:
	rm -f codex-rotator

# Install binary + enable systemd user service (runs as current user, no sudo for service)
install: build
	sudo cp codex-rotator /usr/local/bin/codex-rotator
	mkdir -p $(HOME)/.config/systemd/user
	cp packaging/systemd/codex-rotator.service $(HOME)/.config/systemd/user/
	systemctl --user daemon-reload
	systemctl --user enable --now codex-rotator
	@echo ""
	@echo "✓ codex-rotator installed and running"
	@echo "  Status : systemctl --user status codex-rotator"
	@echo "  Logs   : journalctl --user -u codex-rotator -f"

uninstall:
	systemctl --user disable --now codex-rotator 2>/dev/null || true
	rm -f $(HOME)/.config/systemd/user/codex-rotator.service
	systemctl --user daemon-reload
	sudo rm -f /usr/local/bin/codex-rotator
	@echo "✓ codex-rotator uninstalled"
