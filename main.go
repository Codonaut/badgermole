package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func main() {
	g, err := NewClient("https://api.github.com/", "Basic Q29kb25hdXQ6ckJDd0J2ZEhwdmlBQndUWENtY3RnUEcz", "Codonaut")
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	ctx := context.Background()
	p, e := g.ListPullRequests(ctx, "Codonaut", "badgermole", nil)
	fmt.Printf("%+v\n%+v\n", p, e)
}

type Client struct {
	accessToken string
	client      *http.Client
	username    string

	BaseURL *url.URL
}

func NewClient(baseURL, accessToken, username string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(u.Path, "/") {
		return nil, fmt.Errorf("baseURL must have a trailing slash, but %q does not", baseURL)
	}

	return &Client{
		accessToken: accessToken,
		username:    username,
		client:      http.DefaultClient,

		BaseURL: u,
	}, nil
}

func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	u, err := c.BaseURL.Parse(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "vnd.github.v3+json")
	req.Header.Add("User-Agent", "autoship")
	req.Header.Add("Authorization", c.accessToken)

	return req, nil
}

func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) error {
	res, err := c.client.Do(req.WithContext(ctx))
	if err != nil {
		select {
		case <-ctx.Done():
			// If we got an error, and the context has been canceled,
			// the context's error is probably more useful.
			return ctx.Err()
		default:
			return err
		}
	}
	defer res.Body.Close()

	if v == nil {
		return nil
	}

	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		if err != io.EOF {
			return err
		}
	}

	return nil
}

type ListPullRequestOptions struct {
	State     string // Either open, closed, or all to filter by state. Default: open.
	Head      string // Filter pulls by head user and branch name in the format of user:ref-name.
	Base      string // Filter pulls by base branch name.
	Sort      string // Can be either created, updated, popularity, or long-running. Default: created.
	Direction string // Can be either asc or desc. Default: desc when sort is created or not specified, otherwise desc.
}

func (o *ListPullRequestOptions) String() string {
	if o == nil {
		return ""
	}

	v := url.Values{}

	if o.State != "" {
		v.Add("state", o.State)
	}
	if o.Head != "" {
		v.Add("head", o.Head)
	}
	if o.Base != "" {
		v.Add("base", o.Base)
	}
	if o.Sort != "" {
		v.Add("sort", o.Sort)
	}
	if o.Direction != "" {
		v.Add("direction", o.Direction)
	}

	return v.Encode()
}

func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, opts *ListPullRequestOptions) ([]PullRequest, error) {
	u := fmt.Sprintf("repos/%s/%s/pulls?%s", owner, repo, opts)
	req, err := c.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var prs []PullRequest
	if err := c.Do(ctx, req, &prs); err != nil {
		return nil, err
	}

	return prs, nil
}

func (c *Client) ListComments(ctx context.Context, owner, repo string, number int) ([]Comment, error) {
	u := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, number)

	req, err := c.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var comments []Comment
	if err := c.Do(ctx, req, comments); err != nil {
		return nil, err
	}

	return comments, nil
}

func (c *Client) CommentOnPullRequest(ctx context.Context, owner, repo string, number int, comment string) error {
	u := fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, repo, number)

	body, err := json.Marshal(&CommentInput{Body: comment})
	if err != nil {
		return err
	}

	req, err := c.NewRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	return c.Do(ctx, req, nil)
}

type User struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
}

type Hook struct {
	ID   int    `json:"id"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

type Label struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Object struct {
	Ref        string     `json:"ref"`
	SHA        string     `json:"sha"`
	User       User       `json:"user"`
	Repository Repository `json:"repo"`
}

type Branch struct {
	Name   string `json:"name"`
	Commit Commit `json:"commit"`
}

// SanitizeName transforms a git branch name to a valid target group and ECS service name.
//
// A target group name may contain 32 alphanumeric characters and hyphens and
// cannot begin or end with a hyphen.
// https://docs.aws.amazon.com/cli/latest/reference/elbv2/create-target-group.html#options
//
// An ECS service name may contain 255 alphanumeric characters, hyphens, and underscores.
// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service_definition_parameters.html
func (b Branch) SanitizeName() string {
	reg := regexp.MustCompile("[^a-zA-Z0-9-]+")
	sanitized := reg.ReplaceAllString(b.Name, "")
	sanitized = sanitized[:min(32, len(sanitized))]
	return strings.Trim(sanitized, "-")
}

func (b Branch) String() string {
	return b.Name + "@" + b.Commit.SHA
}

type Commit struct {
	SHA string `json:"sha"`
}

// StatusEvent is triggered when the status of a Git commit changes.
type StatusEvent struct {
	SHA         string     `json:"sha"`
	State       string     `json:"state"`
	Description string     `json:"description"`
	TargetURL   string     `json:"target_url"`
	Branches    []Branch   `json:"branches"`
	Repository  Repository `json:"repository"`
}

type Repository struct {
	Name          string `json:"name"`
	Owner         User   `json:"owner"`
	IsFork        bool   `json:"fork"`
	DefaultBranch string `json:"default_branch"`
}

// PullRequestEvent is triggered when a pull request is created, assigned,
// unassigned, labeled, unlabeled, opened, edited, closed, reopened,
// synchronized, or removed.
//
// https://developer.github.com/v3/activity/events/types/#pullrequestevent
type PullRequestEvent struct {
	Action      string      `json:"action"`
	PullRequest PullRequest `json:"pull_request"`
	Label       *Label      `json:"label"` // Non-nil when `Action` is "labeled" or "unlabeled".
}

type PullRequest struct {
	ID             int     `json:"id"`
	Labels         []Label `json:"labels"`
	Number         int     `json:"number"`
	State          string  `json:"state"`
	Title          string  `json:"title"`
	URL            string  `json:"url"`
	User           User    `json:"user"`
	MergeCommitSHA string  `json:"merge_commit_sha"`
	Head           Object  `json:"head"`
	Base           Object  `json:"base"`
}

func (r PullRequest) ContainsLabel(name string) bool {
	for _, l := range r.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

type PullRequests []PullRequest

func (rr PullRequests) ContainsLabel(name string) (PullRequest, bool) {
	for _, r := range rr {
		if r.ContainsLabel(name) {
			return r, true
		}
	}

	return PullRequest{}, false
}

type PingEvent struct {
	Zen  string `json:"zen"`
	Hook Hook   `json:"hook"`
}

// DeleteEvent is triggered when a ref (branch or tag) is deleted.
// https://developer.github.com/v3/activity/events/types/#deleteevent
type DeleteEvent struct {
	RefType string `json:"ref_type"`
	Ref     string `json:"ref"`
}

type CommentInput struct {
	Body string `json:"body"`
}

type Comment struct {
	Id       string `json:"id"`
	CommitId string `json:"commit_id"`
	User     User   `json:"user"`
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
