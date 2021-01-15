package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/go-github/github"
)

func main() {
	fmt.Println("main")
	http.HandleFunc("/", HookHandler)
	log.Fatal(http.ListenAndServe(":9999", nil))

	// ctx := context.Background()
	// ts := oauth2.StaticTokenSource(
	// 	&oauth2.Token{AccessToken: "d228fb13f42f8bab2d098436038f86e1f75f8552"},
	// )
	// tc := oauth2.NewClient(ctx, ts)

	// client := github.NewClient(tc)

	// // list all repositories for the authenticated user

	// repos, _, err := client.Repositories.List(ctx, "", nil)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// for repos
	// fmt.Println(repos)
}

// HookHandler parses GitHub webhooks and sends an update to corresponding channel
func HookHandler(w http.ResponseWriter, r *http.Request) {
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
	log.Printf("received event: %v\n", event)

	switch e := event.(type) {
	case *github.PullRequestEvent:
		log.Printf("received PullRequestEvent: %v\n", e)
	case *github.PullRequestReviewEvent:
		log.Printf("received PullRequestReviewEvent: %v\n", e)
	case *github.PullRequestReviewCommentEvent:
		log.Printf("received PullRequestReviewCommentEvent: %v\n", e)
	default:
		log.Printf("unknown event type %s\n", github.WebHookType(r))
		return
	}
}
