# Atalhos de desenvolvimento.  make help
.PHONY: help build teste test cover lint release

help:  ## lista os comandos
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | sed 's/:.*## /\t/'

build:  ## compila ./termux-setup
	go build -o termux-setup ./cmd/termux-setup

teste: build  ## abre a interface no modo simulação
	./termux-setup --teste

test:  ## roda todos os testes
	go test ./...

cover:  ## testes com cobertura
	go test -coverprofile=cover.out ./... && go tool cover -func=cover.out | tail -n 1

lint:  ## formatação, vet e shellcheck
	test -z "$$(gofmt -l .)"
	go vet ./...
	shellcheck -S warning install.sh scripts/*.sh
	shellcheck -s bash -S warning internal/ubuntu/scripts/*.sh

release:  ## gera dist/ (ex.: make release V=v6.0.0)
	scripts/build-release.sh $(V)
