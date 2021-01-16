There's a cute tool called [Atlantis that](https://www.runatlantis.io) runs Terraform in response to events on pull requests. Without going into the nitty gritty and 10 thousand configuration details the core workflow is dead simple - when a PR is open or synchronized (HEAD commit changes), it checks out your code for the HEAD of that PR and uses it to "dry-run" (terraform plan) against the corresponding Terraform project, then posts the result as a comment. When you put a comment like "atlantis apply", it does pretty much the same thing, only this time it runs "for real" ("terraform apply"). Then it posts the result as a comment.

We'd like you to build a similar convenience wrapper for another IaC tool, [Pulumi](https://www.pulumi.com) - let's be original and call it Pullantis (see what we did there?). Pullantis will do the same thing, just for Pulumi. In your solution we will assume that you have all the Pulumi binaries available on the server, so no need to manage versions. We'll also assume that your handler manages a single Pulumi project (they call them Stacks, just as we do), so you can hardcode all the details. We'll also assume that you only need to support GitHub and instead of polling their API you will respond to webhooks, though how you receive them is up to you.

So, when a PR is open, you run Pulumi's equivalent of `terraform plan` against the source and post the outcome as a PR review. For a positive outcome (no errors) you approve the PR, for a negative (anything goes wrong) - you request changes. If there are new commits on the PR, you do the same. When someone types in `pullantis apply`, you run the Pulimi's equivalent of `terraform apply` and post the result as a regular comment. Don't worry about any preconditions - the comment is all you need, whether the code looks good or not. It's also OK to apply multiple times from the same branch.

We want your solution to implement some form of queuing mechanism (in-memory is fine) to make sure that you eventually serve all the received requests, but you serve them one at the time, in the same order you received them. This request queue is the only state we want you to keep. Other than that, the tool is expected to be stateless.

Please feel free to make as many assumptions, take as many shortcuts, and hardcode as many things as necessary to make this project a quick hack rather than a masterpiece of software engineering. No unit tests required either, we'll just test it  on a real repo. That said, we actually want you to run Pulumi and pass the output to GitHub.


# Setup

```
export GITHUB_AUTH_TOKEN=your_acces_token
go run .
ngrok http 9999
```

Paste Ngrok URL in webhook. The webhook secret is `supersecretstring`