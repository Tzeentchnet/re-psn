package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"html/template"
	"strings"
)

// Assets are embedded so deployments remain a single binary. Only these
// compile-time files enter the trusted CSS/JS contexts; server data stays escaped.
//
//go:embed status.html
var statusHTMLRaw string

//go:embed status.css
var statusCSSRaw string

//go:embed status.js
var statusJSRaw string

// HTML parsers normalize line endings before CSP hashes are checked. Normalize
// both the rendered assets and their hashes, including Windows CRLF checkouts.
func statusLF(source string) string {
	return strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n")
}

var (
	statusHTML = statusLF(statusHTMLRaw)
	statusCSS  = statusLF(statusCSSRaw)
	statusJS   = statusLF(statusJSRaw)
)

func statusHash(source string) string {
	hash := sha256.Sum256([]byte(statusLF(source)))
	return "'sha256-" + base64.StdEncoding.EncodeToString(hash[:]) + "'"
}

var statusPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; " +
	"connect-src 'self'; style-src " + statusHash(statusCSS) + "; script-src " + statusHash(statusJS)

var page = template.Must(template.New("status").Funcs(template.FuncMap{
	"style":   func() template.CSS { return template.CSS(statusCSS) },
	"script":  func() template.JS { return template.JS(statusJS) },
	"version": func() string { return version },
}).Parse(statusHTML))
