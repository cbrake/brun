#!/usr/bin/env bash
# this file should be sourced (.), not run as a script

brun_format() {
  prettier --write "*.md" || return 1
  gofmt -w . || return 1
}

brun_format_check() {
  prettier --check "*.md" || return 1
  gofmt -l . | grep -q '.' && { echo "gofmt: files need formatting:"; gofmt -l .; return 1; }
}
