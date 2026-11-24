package sys

import (
	"archive/zip"
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"
)

// Fetcher baixa uma URL para um arquivo.
//
// O download de verdade usa o curl do Termux (e não a rede do Go) de
// propósito: binários Go estáticos no Android não acham o DNS nem os
// certificados do sistema, e o curl do Termux já vem configurado.
type Fetcher interface {
	Fetch(ctx context.Context, url, dest string) error
}

// CurlFetcher baixa com curl.
type CurlFetcher struct {
	Run     Runner
	Timeout time.Duration // 0 = 180s
}

func (f CurlFetcher) Fetch(ctx context.Context, url, dest string) error {
	t := f.Timeout
	if t == 0 {
		t = 180 * time.Second
	}
	return f.Run.Run(ctx, C("curl", "-fsSL", "--retry", "2", "--max-time", fmt.Sprint(int(t.Seconds())), "-o", dest, url), nil)
}

// DryFetcher gera arquivos plausíveis em vez de baixar (modo --teste):
// Release de espelho com data aleatória, zip de fonte, script de instalação.
type DryFetcher struct{}

func (DryFetcher) Fetch(ctx context.Context, url, dest string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(80+rand.IntN(400)) * time.Millisecond):
	}
	switch {
	case strings.Contains(url, "/dists/"):
		age := time.Duration(rand.IntN(70)) * time.Hour
		body := "Origin: Termux\nDate: " + time.Now().Add(-age).UTC().Format(time.RFC1123) + "\n"
		return os.WriteFile(dest, []byte(body), 0o644)
	case strings.HasSuffix(url, ".zip"):
		return fakeFontZip(dest)
	default:
		return os.WriteFile(dest, []byte("#!/bin/sh\necho '[teste] instalador'\n"), 0o755)
	}
}

func fakeFontZip(dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	w, err := z.Create("fonts/ttf/JetBrainsMono-Regular.ttf")
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte("FONTE FALSA")); err != nil {
		return err
	}
	return z.Close()
}
