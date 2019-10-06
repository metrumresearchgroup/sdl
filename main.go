package main

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

type fileDl struct {
	URLs      []string `json:"url,omitempty"`
	FileName  string   `json:"file_name,omitempty"`
	Overwrite bool     `json:"overwrite,omitempty"`
}

type dlResult struct {
	Settings           fileDl
	URL                string
	SuccessfulDownload bool
	AlreadyExists      bool
	Error              error
}

var fileDls []fileDl
var dlResults []dlResult
var rootDir string
var fs = afero.NewOsFs()
var jsonFile string
var cranStructure = []string{
	"src/contrib",
	"bin/macosx/el-capitan/contrib/3.5",
	"bin/macosx/el-capitan/contrib/3.6",
	"bin/windows/contrib/3.5",
	"bin/windows/contrib/3.6",
}

func init() {
	flag.StringVar(&jsonFile, "jsonFile", "", "jsonfile with files to dl")
	flag.StringVar(&rootDir, "dir", ".", "directory to download the files")
	flag.Parse()
}
func main() {
	jsonExpanded, _ := homedir.Expand(jsonFile)
	jsonBytes, err := afero.ReadFile(fs, jsonExpanded)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":  err,
			"file": jsonFile,
		}).Fatal("could not read json file")
	}
	err = json.Unmarshal(jsonBytes, &fileDls)
	if err != nil {
		log.WithFields(logrus.Fields{
			"err":  err,
			"file": jsonFile,
		}).Fatal("could not parse json file")
	}
	if rootDir != "." {
		err := fs.MkdirAll(rootDir, 0777)
		// mkdirall wil return nil if already exists
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
				"dir": rootDir,
			}).Fatal("could not create root dir to download files")

		}
	}
	for _, fp := range cranStructure {
		err := fs.MkdirAll(filepath.Join(rootDir, fp), 0777)
		// mkdirall wil return nil if already exists
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":  err,
				"dir":  rootDir,
				"cran": fp,
			}).Fatal("could not create cranlike dir to download files")

		}
	}
	sem := make(chan struct{}, 30)
	done := make(chan dlResult, len(dlResults))
	// wg := sync.WaitGroup{}
	for _, d := range fileDls {
		// wg.Add(1)
		go func(fdl fileDl, doneCh chan dlResult) {
			sem <- struct{}{}
			defer func() {
				<-sem
				// wg.Done()
			}()
			fpath := filepath.Join(rootDir, fdl.FileName)
			maybeDownloadAndSave(fdl, fpath, doneCh)

		}(d, done)
	}
	successDownloads := 0
	filesAvailable := 0
	errors := 0
	for i := 0; i < len(fileDls); i++ {
		result := <-done
		if result.Error != nil {
			errors++
			log.WithFields(logrus.Fields{
				"error": err,
				"urls":  strings.Join(result.Settings.URLs, "||"),
				"name":  result.Settings.FileName,
			}).Warn("no suitable download")
		}
		if result.SuccessfulDownload {
			successDownloads++
		}
		if result.SuccessfulDownload || result.AlreadyExists {
			filesAvailable++
		}
		dlResults = append(dlResults, result)
	}
	log.WithFields(logrus.Fields{
		"downloads": successDownloads,
		"files":     filesAvailable,
		"errors":    errors,
	}).Info("results")
}
