package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		fmt.Println("method not allowed")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println("----------------------")
	fmt.Println(string(body))

	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

// supersecretstring
func main() {
	fmt.Println("main")
	http.HandleFunc("/", handler)
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
