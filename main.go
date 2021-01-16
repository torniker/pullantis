package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
)

type PullRequest struct {
	SHA string
	URL string
}

func main() {
	prChan := make(chan PullRequest)
	go listener(prChan)
	http.HandleFunc("/", HookHandler(prChan))
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func listener(prChan chan PullRequest) {
	for {
		select {
		case p := <-prChan:
			downloadURL := fmt.Sprintf("%s/archive/%s.zip", p.URL, p.SHA)
			resp, err := http.Get(downloadURL)
			if err != nil {
				log.Printf("err: %s", err)
			}

			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				continue
			}
			zipFile := fmt.Sprintf("./tmp/%s.zip", p.SHA)
			out, err := os.Create(zipFile)
			if err != nil {
				log.Printf("error creating file: %s", err)
			}
			defer out.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Printf("error copying data into file: %s", err)
			}
			_, err = Unzip(zipFile, fmt.Sprintf("./tmp/%s", p.SHA))
			if err != nil {
				log.Printf("error unziping: %s", err)
			}
			log.Printf("listener got event: %#v, url: %s\n", p.SHA, downloadURL)
		}
	}
}

// HookHandler parses GitHub webhooks and sends an update to corresponding channel
func HookHandler(prChan chan<- PullRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := github.ValidatePayload(r, []byte("supersecretstring"))
		if err != nil {
			log.Printf("error validating request body: err=%s\n", err)
			return
		}
		defer r.Body.Close()
		event, err := github.ParseWebHook(github.WebHookType(r), payload)
		if err != nil {
			log.Printf("could not parse webhook: err=%s\n", err)
			return
		}
		// log.Printf("received event: %v\n", event)
		switch e := event.(type) {
		case *github.PullRequestEvent:
			prChan <- PullRequest{
				URL: e.GetRepo().GetHTMLURL(),
				SHA: *e.PullRequest.Head.SHA,
			}
		case *github.PullRequestReviewCommentEvent:
			log.Printf("received PullRequestReviewCommentEvent: %v\n", e)
		default:
			log.Printf("unknown event type %s\n", github.WebHookType(r))
			return
		}
	}
}

// Unzip does what is says
func Unzip(src string, dest string) ([]string, error) {
	var filenames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()
	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)
		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}
		filenames = append(filenames, fpath)
		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}
		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		_, err = io.Copy(outFile, rc)
		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()
		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
