package hook

import (
	"errors"
	"net/http"
)

// ErrSecretValidationFailed is returned when the webhook secret/token
// provided in the request does not match the expected secret.
// Callers can use errors.Is to check for this sentinel error.
var ErrSecretValidationFailed = errors.New("webhook secret validation failed")

// ErrCommentNotPullRequest is returned when a comment event is received
// but the comment is not associated with a pull request (e.g., issue comment).
var ErrCommentNotPullRequest = errors.New("comment is not associated with a pull request")

// ErrUnsupportedEvent is returned when a webhook parser receives an event
// type header that it does not recognize. Callers can use errors.Is to
// detect this condition and skip processing.
var ErrUnsupportedEvent = errors.New("unsupported webhook event type")

// ErrUnsupportedAction is returned when a webhook payload contains an
// action value that the parser does not handle (e.g., a PR "closed" event
// when only "opened" and "synchronize" are supported).
var ErrUnsupportedAction = errors.New("unsupported webhook action")

type (
	WebHook interface {
		Repository() Repository
	}
	WebhookService interface {
		Parse(req *http.Request, fn SecretFunc) (WebHook, error)
	}

	PushHook struct {
		Ref     string
		Repo    Repository
		Before  string
		After   string
		Commit  Commit
		Sender  User
		Commits []Commit
	}
	// BranchHook represents a branch or tag event,
	// eg create and delete github event types.
	BranchHook struct {
		Ref    Reference
		Repo   Repository
		Sender User
	}

	PullRequestHook struct {
		Action      string
		Repo        Repository
		TargetRepo  Repository
		PullRequest PullRequest
		Sender      User
	}

	PullRequestCommentHook struct {
		Action      string
		Repo        Repository
		TargetRepo  Repository
		PullRequest PullRequest
		Comment     Comment
		Sender      User
	}

	SecretFunc func(webhook WebHook) (string, error)
)

func (h *PushHook) Repository() Repository               { return h.Repo }
func (h *BranchHook) Repository() Repository             { return h.Repo }
func (h *PullRequestHook) Repository() Repository        { return h.Repo }
func (h *PullRequestCommentHook) Repository() Repository { return h.Repo }
