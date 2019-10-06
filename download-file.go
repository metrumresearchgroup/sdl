package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/afero"
)

// downloadFileData should return the file data as well as the url string
// for which URL was successful in downloading
func downloadFileData(URLs []string) ([]byte, string, error) {
	var globalerr error
	for _, URL := range URLs {
		response, err := http.Get(URL)
		if err != nil {
			// I guess we'll just continue
			globalerr = err
			continue
		}
		// don't close body until check err is non-nil
		// can't defer as in a loop and could have multiple
		// responses
		// https://stackoverflow.com/questions/33238518/what-could-happen-if-i-dont-close-response-body-in-golang
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			continue
		}
		var data bytes.Buffer
		_, err = io.Copy(&data, response.Body)
		response.Body.Close()
		// if the Copy error fails, it means some other issue beyond
		// getting a correct response, so should probably actually error
		// rather than trying to keep going
		if err != nil {
			return nil, URL, fmt.Errorf("error copying data from response: %w")
		}
		return data.Bytes(), URL, nil
	}
	return nil, "", fmt.Errorf("no suitable url found, last error: %w", globalerr)
}

func maybeDownloadAndSave(d fileDl, fpath string, doneCh chan dlResult) {
	if fileExists(fpath) {
		doneCh <- dlResult{
			Settings:           d,
			SuccessfulDownload: false,
			AlreadyExists:      true,
			Error:              nil,
		}
		return
	}
	fileContents, url, err := downloadFileData(d.URLs)
	if err != nil {
		doneCh <- dlResult{
			Settings:           d,
			URL:                url,
			SuccessfulDownload: false,
			AlreadyExists:      false,
			Error:              err,
		}
		return
	}
	err = afero.WriteFile(fs, fpath, fileContents, 0777)
	if err != nil {
		doneCh <- dlResult{
			Settings:           d,
			SuccessfulDownload: false,
			AlreadyExists:      false,
			Error:              err,
		}
		return
	}
	doneCh <- dlResult{
		Settings:           d,
		SuccessfulDownload: true,
		AlreadyExists:      false,
		Error:              nil,
	}
}
