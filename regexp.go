package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strings"
)

type RegCat struct {
	Regexp   *regexp.Regexp
	Category string
}

var regexps struct {
	Filename       []RegCat //1
	Content        []RegCat //2
	IgnoreContent  []RegCat //3
	IgnoreFilename []RegCat //4
}

func loadRegexps(filename string) {
	fileHandle, err := os.Open(filename)
	testErrDie("Unable to open regexp file", err)
	scanner := bufio.NewScanner(fileHandle)

	section := 0
	category := "none"
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, ";section") {
			category = "none"
			if strings.HasSuffix(line, "ignore-filename") {
				section = 4
			} else if strings.HasSuffix(line, "ignore-content") {
				section = 3
			} else if strings.HasSuffix(line, "content") {
				section = 2
			} else if strings.HasSuffix(line, "filename") {
				section = 1
			} else {
				log.Fatalf("Error in regexp file at line: %d\n", lineNumber)
			}
		} else if strings.HasPrefix(line, ";category") {
			v := strings.SplitN(line, " ", 2)
			category = v[1]
		} else {
			reg := regexp.MustCompile(line)
			switch section {
			case 1:
				regexps.Filename = append(regexps.Filename, RegCat{Regexp: reg, Category: category})
			case 2:
				regexps.Content = append(regexps.Content, RegCat{Regexp: reg, Category: category})
			case 3:
				regexps.IgnoreContent = append(regexps.IgnoreContent, RegCat{Regexp: reg, Category: category})
			case 4:
				regexps.IgnoreFilename = append(regexps.IgnoreFilename, RegCat{Regexp: reg, Category: category})
			}
		}
	}
}
