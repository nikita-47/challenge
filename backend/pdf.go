package main

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractPDFText reads a PDF file and returns the text content page by page.
func ExtractPDFText(filepath string) (pages []string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("PDF parse error: %v", r)
		}
	}()

	f, r, err := pdf.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	totalPages := r.NumPage()
	pages = make([]string, 0, totalPages)

	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			pages = append(pages, "")
			continue
		}

		content, err := page.GetPlainText(nil)
		if err != nil {
			// Non-fatal: include empty string for this page.
			pages = append(pages, "")
			continue
		}

		pages = append(pages, strings.TrimSpace(content))
	}

	return pages, nil
}
