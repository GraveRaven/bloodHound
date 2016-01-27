package main

import (
	"fmt"
	"sync"
)

type FileReport struct {
	Name     string
	Path     string
	Reason   int //1 = filename, 2 = content
	Category string
	Regexp   string
}

func (r FileReport) String() string {
	var reason string
	switch r.Reason {
	case 1:
		reason = "filename"
	case 2:
		reason = "content"
	default:
		reason = "unknown"
	}

	return fmt.Sprintf("%s|%s|%s|%s%s", reason, r.Category, r.Regexp, r.Path, r.Name)
}

func createReport(rc chan FileReport, wg *sync.WaitGroup) {
	defer wg.Done()
	categories := make(map[string][]FileReport)
	for r := range rc {
		categories[r.Category] = append(categories[r.Category], r)
	}

	for _, v := range categories {
		for _, r := range v {
			fmt.Println(r)
		}
	}
}
