// fileFinder project main.go
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wblakecaldwell/profiler"
)

var config struct {
	MaxThreads  int
	MaxFileSize int64 //File size in bytes
	WaitDelay   int
}

func loadConfig(filename string) {
	confFile, err := ioutil.ReadFile(filename)
	testErrDie("Error opening config file", err)

	s := bufio.NewScanner(bytes.NewReader(confFile))
	lineNr := 0
	for s.Scan() {
		lineNr += 1

		if s.Text() == "" || strings.HasPrefix(s.Text(), "#") {
			continue
		}
		line := strings.Split(s.Text(), "=")

		if len(line) != 2 {
			log.Fatalf("Error parsing config at line %d\n", lineNr)
		}

		if line[0] == "threads" && config.MaxThreads == 0 {
			v, err := strconv.Atoi(line[1])
			testErrDie("Error parsing threads int value", err)
			config.MaxThreads = v
		} else if line[0] == "maxSize" && config.MaxFileSize == 0 {
			v, err := ToBytes(line[1])
			testErrDie("Error parsing maxSize", err)
			config.MaxFileSize = v
			fmt.Printf("Max size: %d\n", v)
		}
	}

	if config.MaxThreads == 0 {
		config.MaxThreads = 4
	}
	if config.MaxFileSize == 0 {
		config.MaxFileSize, _ = ToBytes("10MB")
	}
}

func testFilename(filename string) (RegCat, error) {
	for _, reg := range regexps.Filename {
		if reg.Regexp.FindStringIndex(filename) != nil {
			return reg, nil
		}
	}

	return RegCat{}, fmt.Errorf("No match")
}

func openOfficeFile(path string) (content []byte, err error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return content, fmt.Errorf("Unable to unzip %s: %v", path, err)
	}

	defer r.Close()

	fileNames := []string{"WordDocument", "content.xml", "document.xml"}

	for _, f := range r.File {
		if f.FileInfo().IsDir() == false {
			for _, n := range fileNames {
				if filepath.Base(f.Name) == n {
					rc, err := f.Open()
					if err != nil {
						return content, fmt.Errorf("%v\n", err)
					}
					defer rc.Close()
					content, err = ioutil.ReadAll(rc)
					return content, err
				}
			}
		}
	}

	return content, nil
}

func testFile(fileinfo os.FileInfo, fullpath string, reportChan chan FileReport) {

	currDir := filepath.Dir(fullpath) + string(filepath.Separator)

	var content []byte
	var err error

	officeExt := []string{"doc", "docx", "xls", "xlsx", "ods", "odt"}

	for _, ext := range officeExt {
		if strings.HasSuffix(fullpath, ext) {
			content, err = openOfficeFile(fullpath)
			//testErrLog("", err)
			break
		}
	}

	if len(content) == 0 {

		content, err = ioutil.ReadFile(fullpath)
		if testErrLog(fmt.Sprintf("Unable to read %s", fullpath), err) == false {
			return
		}
	}

	for _, reg := range regexps.Content {
		if reg.Regexp.FindIndex(content) != nil {
			reportChan <- FileReport{
				Name:     fileinfo.Name(),
				Path:     currDir,
				Reason:   2,
				Category: reg.Category,
				Regexp:   reg.Regexp.String(),
			}
		}
	}
}

func newFilePath(currDir string, newFile string) string {
	return fmt.Sprintf("%s%s", currDir, newFile)
}

func newDirPath(currDir string, newDir string) string {
	return fmt.Sprintf("%s%s%c", currDir, newDir, os.PathSeparator)
}

type nrDir struct {
	sync.Mutex
	nr int
}

func (n *nrDir) Add() {
	n.Lock()
	n.nr += 1
	n.Unlock()
}

var nrOfDirs nrDir

type nrFile struct {
	sync.Mutex
	nr int
}

func (n *nrFile) Add() {
	n.Lock()
	n.nr += 1
	n.Unlock()
}

var nrOfFiles nrFile

func readDir(fileh *os.File, dispChan chan string, reportChan chan FileReport) {

	fileInfo, err := fileh.Readdir(0)
	if testErrLog(fmt.Sprintf("Error reading %s", fileh.Name()), err) {
		return
	}

INFOLOOP:
	for _, f := range fileInfo {
		if f.Name() == "" { //Bugfix for when filenames contain strange characters
			continue
		}
		if f.IsDir() {
			nrOfDirs.Add()
			dispChan <- newDirPath(fileh.Name(), f.Name())
		} else if f.Mode().IsRegular() {
			nrOfFiles.Add()

			for _, v := range regexps.IgnoreFilename {
				if v.Regexp.FindStringIndex(f.Name()) != nil {
					continue INFOLOOP
				}
			}

			if reg, err := testFilename(f.Name()); err == nil {
				reportChan <- FileReport{
					Name:     f.Name(),
					Path:     fileh.Name(),
					Reason:   1,
					Category: reg.Category,
					Regexp:   reg.Regexp.String(),
				}
			} else if f.Size() <= config.MaxFileSize {
				for _, v := range regexps.IgnoreContent {
					if v.Regexp.FindStringIndex(f.Name()) != nil {
						continue INFOLOOP
					}
				}
				dispChan <- newFilePath(fileh.Name(), f.Name())
			}
		} else {
			log.Printf("Unknown file type: %v\n", newFilePath(fileh.Name(), f.Name()))
		}
	}
}

func worker(workChannel chan string, reportChan chan FileReport, dispChan chan string, wg *sync.WaitGroup) {

	for filename := range workChannel {

		fileh, err := os.Open(filename)
		if testErrLog(fmt.Sprintf("Error opening %s", filename), err) {
			continue
		}

		stat, err := fileh.Stat()
		if testErrLog(fmt.Sprintf("Unable to stat %s", filename), err) {
			continue
		}

		if stat.IsDir() {
			readDir(fileh, dispChan, reportChan)
		} else {
			testFile(stat, fileh.Name(), reportChan)
		}

		fileh.Close()
	}
	wg.Done()
}

func dispatcher(dispChan chan string, workChannel chan string) {
	queue := NewQueueMutex()

	go func(q *QueueMutex, c chan string) {
		for job := range c {
			q.Push(job)
		}
	}(queue, dispChan)

	for v := queue.Pop(); v != nil; v = queue.Pop() {
		workChannel <- v.(string)
	}

	close(workChannel)
	close(dispChan)
}

func myUsage() {
	fmt.Printf("Usage: %s [OPTIONS] <path>\n\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {

	flagWorker := flag.Int("workers", 8, "Number of concurrent workers")
	flagMaxSize := flag.String("max", "10MB", "Max size of files to scan")
	flagWait := flag.Int("wait", 5, "Wait delay before completion")
	//flagConfig := flag.String("config", "config.cfg", "Config file")
	flagDebug := flag.Bool("debug", false, "Debugging")
	flag.Parse()

	if flag.NArg() != 1 {
		myUsage()
		os.Exit(1)
	}

	startDir := filepath.FromSlash(flag.Arg(0))

	if strings.HasSuffix(startDir, string(filepath.Separator)) == false {
		startDir += string(filepath.Separator)
	}

	if *flagDebug {
		profiler.AddMemoryProfilingHandlers()
		profiler.StartProfiling()
		go http.ListenAndServe(":6060", nil)
	}

	//loadConfig(*flagConfig)
	var err error
	config.MaxFileSize, err = ToBytes(*flagMaxSize)
	testErrDie("Error parsing max size", err)

	config.MaxThreads = *flagWorker
	config.WaitDelay = *flagWait

	loadRegexps("regexps.cfg")

	reportChan := make(chan FileReport)
	var reportWg sync.WaitGroup
	reportWg.Add(1)
	go createReport(reportChan, &reportWg)

	workChannel := make(chan string)
	dispChan := make(chan string, 3)

	var workerWg sync.WaitGroup
	workerWg.Add(config.MaxThreads)

	now := time.Now()
	for i := 0; i < config.MaxThreads; i++ {
		go worker(workChannel, reportChan, dispChan, &workerWg)
	}

	go dispatcher(dispChan, workChannel)
	dispChan <- startDir
	workerWg.Wait()
	fmt.Println("Dirs:", nrOfDirs.nr)
	fmt.Println("Files:", nrOfFiles.nr)
	close(reportChan)
	reportWg.Wait()
	fmt.Println(time.Since(now))
}
