.PHONY: build test clean install \
       build-lab-image rebuild-lab-image \
       lab-start lab-exec lab-claude lab-attach lab-list lab-stop lab-cleanup lab-doctor \
       help

BIN := ./claudeup-lab

##@ General

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z_-]+:.*?## / { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Development

build: ## Build the claudeup-lab binary
	go build -o claudeup-lab ./cmd/claudeup-lab

test: ## Run all tests
	go test ./... -v

clean: ## Remove built binary
	rm -f claudeup-lab

install: build ## Build and install to ~/.local/bin
	mkdir -p $(HOME)/.local/bin
	cp claudeup-lab $(HOME)/.local/bin/

##@ Lab Management

build-lab-image: ## Build the base lab Docker image
	@docker build -t ghcr.io/claudeup/claudeup-lab:latest embed/

rebuild-lab-image: ## Rebuild the lab image from scratch (no cache)
	@docker build --no-cache -t ghcr.io/claudeup/claudeup-lab:latest embed/

lab-start: build ## Start a lab (PROJECT=path PROFILE=name [BRANCH=name] [NAME=name] [FEATURE=lang])
	@test -n "$(PROJECT)" || (echo "Error: PROJECT required (path to git repo)" && exit 1)
	@test -n "$(PROFILE)" || (echo "Error: PROFILE required (claudeup profile name)" && exit 1)
	@$(BIN) start --project $(PROJECT) --profile $(PROFILE) \
		$(if $(BRANCH),--branch $(BRANCH)) $(if $(NAME),--name "$(NAME)") $(if $(FEATURE),--feature $(FEATURE))

lab-exec: build ## Open a shell in a lab (LAB=name)
	@test -n "$(LAB)" || (echo "Error: LAB required" && exit 1)
	@$(BIN) exec --lab $(LAB)

lab-claude: build ## Run Claude Code in a lab (LAB=name)
	@test -n "$(LAB)" || (echo "Error: LAB required" && exit 1)
	@$(BIN) exec --lab $(LAB) -- claude

lab-attach: build ## Attach VS Code to a lab (LAB=name)
	@test -n "$(LAB)" || (echo "Error: LAB required" && exit 1)
	@$(BIN) open --lab $(LAB)

lab-list: build ## List active labs
	@$(BIN) list

lab-stop: build ## Stop a lab (LAB=name)
	@test -n "$(LAB)" || (echo "Error: LAB required" && exit 1)
	@$(BIN) stop --lab $(LAB)

lab-cleanup: build ## Remove a lab and its worktree (LAB=name)
	@test -n "$(LAB)" || (echo "Error: LAB required" && exit 1)
	@$(BIN) rm --lab $(LAB) --force

lab-doctor: build ## Check system prerequisites
	@$(BIN) doctor
