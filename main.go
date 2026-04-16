package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"mkepub/assets"
	"mkepub/internal/aozora"
	"mkepub/internal/catalog"
	"mkepub/internal/config"
	"mkepub/internal/cover"
	"mkepub/internal/epub"
	"mkepub/internal/fetcher"
	"mkepub/internal/mailer"
	"mkepub/internal/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "cannot create output dir: %v\n", err)
		os.Exit(1)
	}

	csvPath := cfg.CSVPath
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "CSV not found: %s\n", csvPath)
		fmt.Fprintf(os.Stderr, "Place the CSV at that path or set csv_path in ~/.config/mkepub/config.toml\n")
		os.Exit(1)
	}

	authors, err := catalog.Load(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "catalog error: %v\n", err)
		os.Exit(1)
	}

	convertFn := func(work *catalog.Work) (string, error) {
		return convert(work, cfg.OutputDir)
	}

	var mailFn ui.MailFunc
	if cfg.Mail.From != "" && cfg.Mail.To != "" {
		mailFn = func(path string) error {
			return mailer.Send(cfg.Mail, path)
		}
	}

	m := ui.New(authors, convertFn, mailFn, cfg.OutputDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ui error: %v\n", err)
		os.Exit(1)
	}
}

func convert(work *catalog.Work, outputDir string) (string, error) {
	raw, err := fetcher.FetchText(work.TextURL)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}

	_, _, paragraphs, chapters := aozora.Parse(raw)
	title := work.DisplayTitle()
	author := work.AuthorName()

	coverPNG, _ := cover.Generate(title, author, assets.FontData)

	book := &epub.Book{
		Title:      title,
		Author:     author,
		Language:   "ja",
		CoverPNG:   coverPNG,
		Paragraphs: paragraphs,
		Chapters:   chapters,
	}

	data, err := epub.Build(book)
	if err != nil {
		return "", fmt.Errorf("epub build: %w", err)
	}

	filename := catalog.SanitizeFilename(work.AuthorName()) + "_" + catalog.SanitizeFilename(work.DisplayTitle()) + ".epub"
	outPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	return outPath, nil
}

