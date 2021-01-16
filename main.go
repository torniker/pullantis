package main

import (
	"fmt"
	"log"
	"net/http"

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
			// https://github.com/username/projectname/archive/commitshakey.zip
			downloadURL := fmt.Sprintf("%s/%s", p.URL, p.SHA)
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
