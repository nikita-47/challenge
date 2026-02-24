package main

import (
	"regexp"
	"strings"
)

var (
	reCodeBlock  = regexp.MustCompile("(?s)```[a-z]*\n?(.*?)```")
	reCodeInline = regexp.MustCompile("`([^`\n]+)`")
	reBold       = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reHeading    = regexp.MustCompile(`(?m)^#{1,3} (.+)$`)
	reHRule      = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	reBullet     = regexp.MustCompile(`(?m)^(\s*)[*-] `)
)

func renderMarkdown(s string) string {
	s = reCodeBlock.ReplaceAllString(s, "\033[33m$1\033[0m")
	s = reBold.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = reCodeInline.ReplaceAllString(s, "\033[33m$1\033[0m")
	s = reHeading.ReplaceAllString(s, "\033[1m$1\033[0m")
	s = reHRule.ReplaceAllString(s, strings.Repeat("─", 60))
	s = reBullet.ReplaceAllString(s, "$1• ")
	return s
}
