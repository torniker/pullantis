package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	prChan := make(chan *PullRequest)
	go listener(prChan)
	http.HandleFunc("/", HookHandler(prChan))
	log.Fatal(http.ListenAndServe(":9999", nil))
}

func listener(prChan chan *PullRequest) {
	for {
		select {
		case pr := <-prChan:
			err := pr.Process()
			if err != nil {
				log.Printf("error Process: %s", err)
			}
			log.Printf("listener got event: %#v\n", pr.SHA)
		}
	}
}

// HookHandler parses GitHub webhooks and sends an update to corresponding channel
func HookHandler(prChan chan<- *PullRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("webhook called")
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
		// log.Printf("event: %v", event)
		switch e := event.(type) {
		case *github.PullRequestEvent:
			repoName := strings.Split(*e.GetRepo().FullName, "/")
			if *e.Action == "closed" {
				prChan <- &PullRequest{
					Owner:         repoName[0],
					Repo:          repoName[1],
					Number:        *e.GetPullRequest().Number,
					URL:           e.GetRepo().GetHTMLURL(),
					SHA:           *e.PullRequest.Head.SHA,
					ShouldDestroy: true,
				}
				// TODO: some other stuff
				return
			}
			// synchronize
			prChan <- &PullRequest{
				Owner:  repoName[0],
				Repo:   repoName[1],
				Number: *e.GetPullRequest().Number,
				URL:    e.GetRepo().GetHTMLURL(),
				SHA:    *e.PullRequest.Head.SHA,
			}
		case *github.IssueCommentEvent:
			body := *e.Comment.Body
			body = strings.TrimSpace(body)
			if body != "pullantis apply" {
				return
			}
			client := NewGithubClient()
			// ctx context.Context, owner string, repo string, number int
			repoName := strings.Split(*e.GetRepo().FullName, "/")
			pullRequest, _, err := client.PullRequests.Get(context.Background(), repoName[0], repoName[1], *e.Issue.Number)
			if err != nil {
				log.Printf("could not fetch PR: %v\n", err)
				return
			}
			prChan <- &PullRequest{
				Owner:       repoName[0],
				Repo:        repoName[1],
				Number:      *e.Issue.Number,
				URL:         e.GetRepo().GetHTMLURL(),
				SHA:         *pullRequest.Head.SHA,
				ShouldApply: true,
			}
		default:
			log.Printf("unknown event type (%T) %s\n", e, github.WebHookType(r))
			return
		}
	}
}

// PullRequest wrapper
type PullRequest struct {
	Owner         string
	Repo          string
	Number        int
	SHA           string
	URL           string
	ShouldApply   bool
	ShouldDestroy bool
	mu            sync.Mutex
}

func (pr *PullRequest) dir() string {
	return fmt.Sprintf("./tmp/%s", pr.SHA)
}

// Process the eventsdfsafsad
func (pr *PullRequest) Process() error {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	zipFile, err := pr.DownloadRepoZip("./tmp")
	if err != nil {
		return fmt.Errorf("error downloading: %s", err)
	}
	_, err = Unzip(*zipFile, pr.dir())
	if err != nil {
		return fmt.Errorf("error unziping: %s", err)
	}
	if pr.ShouldDestroy {
		return pr.Destroy()
	}
	msg, succeeded := pr.DryRun()
	if pr.ShouldApply && succeeded {
		msg, succeeded = pr.Apply()
	}
	err = pr.CreateReview(msg, succeeded)
	if err != nil {
		return fmt.Errorf("error reviewing PR %s", err)
	}
	return nil
}

// Destroy pulumi destroy
func (pr *PullRequest) Destroy() error {
	// cmd := exec.Command("pulumi", "--cwd", pr.dir(), "destroy", "--non-interactive", "--yes")
	cmd := exec.Command("ls")
	var out bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &out
	_ = cmd.Run()
	if out.Len() > 0 {
		return fmt.Errorf("could not destory: %s", out.String())
	}
	return nil
}

// Apply runs pulumi up for PR
func (pr *PullRequest) Apply() (string, bool) {
	cmd := exec.Command("pulumi", "--cwd", pr.dir(), "up")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	return out.String(), false
}

// DryRun runs pulumi preview for PR
func (pr *PullRequest) DryRun() (string, bool) {
	cmd := exec.Command("pulumi", "--cwd", pr.dir(), "preview")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err == nil
}

// CreateReview for pull request
func (pr *PullRequest) CreateReview(msg string, approve bool) error {
	client := NewGithubClient()
	event := "REQUEST_CHANGES"
	if approve {
		event = "APPROVE"
	}
	newComment := &github.PullRequestReviewRequest{
		Body:     &msg,
		CommitID: &pr.SHA,
		Event:    &event,
	}
	// TODO: there is a limit to create review on github need to handle this.
	_, _, err := client.PullRequests.CreateReview(context.Background(), pr.Owner, pr.Repo, pr.Number, newComment)
	if err != nil {
		if er, ok := err.(*github.ErrorResponse); ok {
			log.Printf("%#v\n", er.Message)
		}
		return fmt.Errorf("counld not comment on pull request (%d): %s", pr.Number, err)
	}
	return nil
}

// DownloadRepoZip downloads repo zip and saves it into dst folder
func (pr *PullRequest) DownloadRepoZip(dst string) (*string, error) {
	downloadURL := fmt.Sprintf("%s/archive/%s.zip", pr.URL, pr.SHA)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("could not fetching repo %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad response status: %d", resp.StatusCode)
	}
	zipFile := fmt.Sprintf("%s/%s.zip", dst, pr.SHA)
	out, err := os.Create(zipFile)
	if err != nil {
		return nil, fmt.Errorf("could not create file: %s", err)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not copy data into file: %s", err)
	}
	return &zipFile, nil
}

// NewGithubClient initializes github client
func NewGithubClient() *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_AUTH_TOKEN")},
	)
	return github.NewClient(oauth2.NewClient(ctx, ts))
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
			return filenames, fmt.Errorf("bad file path: %s", fpath)
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
