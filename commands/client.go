// Package github is a minimal hand-rolled GitHub REST client (net/http only).
package commands

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	UniverseOwner = "typst"
	UniverseRepo  = "packages"
)

type Client struct {
	HTTP    *http.Client
	Token   string // empty = unauthenticated
	BaseURL string // default https://api.github.com
}

func New(token string) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: 30 * time.Second},
		Token:   token,
		BaseURL: "https://api.github.com",
	}
}

func (c *Client) do(method, path string, query map[string]string, body any, out any) error {
	u := strings.TrimSuffix(c.BaseURL, "/") + path
	if len(query) > 0 {
		q := url.Values{}
		for k, v := range query {
			q.Set(k, v)
		}
		u += "?" + q.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, u, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "typkg")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github API %s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode github response %s %s: %w", method, path, err)
	}
	return nil
}

type ContentItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"`
}

// GetContents lists a directory in a repo.
func (c *Client) GetContents(owner, repo, path, ref string) ([]ContentItem, error) {
	var items []ContentItem
	// Single item (file) returns an object; we only call for dirs.
	err := c.do("GET", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, strings.TrimPrefix(path, "/")),
		map[string]string{"ref": ref}, nil, &items)
	if err != nil {
		// try single-object form to surface a clearer error
		var single ContentItem
		err2 := c.do("GET", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, strings.TrimPrefix(path, "/")),
			map[string]string{"ref": ref}, nil, &single)
		if err2 == nil {
			return []ContentItem{single}, nil
		}
		return nil, err
	}
	return items, nil
}

type Pull struct {
	Number  int     `json:"number"`
	Title   *string `json:"title"`
	HTMLURL *string `json:"html_url"`
	URL     string  `json:"url"`
}

func (c *Client) ListOpenPulls() ([]Pull, error) {
	var pulls []Pull
	if err := c.do("GET", fmt.Sprintf("/repos/%s/%s/pulls", UniverseOwner, UniverseRepo),
		map[string]string{"state": "open", "per_page": "100"}, nil, &pulls); err != nil {
		return nil, err
	}
	return pulls, nil
}

type User struct {
	Login string `json:"login"`
}

func (c *Client) CurrentUser() (*User, error) {
	var u User
	if err := c.do("GET", "/user", nil, nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

type Repo struct {
	Name   string `json:"name"`
	Parent *Repo  `json:"parent"`
	Owner  *User  `json:"owner"`
}

func (c *Client) GetRepo(owner, repo string) (*Repo, error) {
	var r Repo
	if err := c.do("GET", fmt.Sprintf("/repos/%s/%s", owner, repo), nil, nil, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

type GitRef struct {
	Ref    string `json:"ref"`
	Object struct {
		SHA string `json:"sha"`
	} `json:"object"`
}

func (c *Client) GetBranchHead(owner, repo, branch string) (string, error) {
	var ref GitRef
	if err := c.do("GET", fmt.Sprintf("/repos/%s/%s/git/ref/heads/%s", owner, repo, branch), nil, nil, &ref); err != nil {
		return "", err
	}
	return ref.Object.SHA, nil
}

type Branch struct {
	Name string `json:"name"`
}

func (c *Client) BranchExists(owner, repo, branch string) (bool, error) {
	var branches []Branch
	if err := c.do("GET", fmt.Sprintf("/repos/%s/%s/branches", owner, repo),
		map[string]string{"per_page": "100"}, nil, &branches); err != nil {
		return false, err
	}
	for _, b := range branches {
		if b.Name == branch {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) DeleteBranch(owner, repo, branch string) error {
	return c.do("DELETE", fmt.Sprintf("/repos/%s/%s/git/refs/heads/%s", owner, repo, branch), nil, nil, nil)
}

func (c *Client) CreateBranch(owner, repo, branch, sha string) error {
	return c.do("POST", fmt.Sprintf("/repos/%s/%s/git/refs", owner, repo), map[string]string{},
		map[string]string{"ref": "refs/heads/" + branch, "sha": sha}, nil)
}

// CreateFile creates/updates a file via Contents API.
func (c *Client) CreateFile(owner, repo, path, message, branch string, content []byte) error {
	return c.do("PUT", fmt.Sprintf("/repos/%s/%s/contents/%s", owner, repo, strings.TrimPrefix(path, "/")),
		nil, map[string]string{
			"message": message,
			"content": base64.StdEncoding.EncodeToString(content),
			"branch":  branch,
		}, nil)
}

type NewPull struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) CreatePull(title, head, base, body string, draft bool) (*NewPull, error) {
	var out NewPull
	if err := c.do("POST", fmt.Sprintf("/repos/%s/%s/pulls", UniverseOwner, UniverseRepo), nil,
		map[string]any{"title": title, "head": head, "base": base, "body": body, "draft": draft}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
