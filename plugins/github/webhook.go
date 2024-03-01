package github

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/iBug/uniAPI/common"
)

type GitPullPayload struct {
	Ref string `json:"ref"`
}

type GitHubWebhook struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Secret string `json:"secret"`
}

var (
	errMissing = errors.New("missing signature")
	errInvalid = errors.New("invalid signature")
	errWrong   = errors.New("bad signature")
)

func validateSignature(sigHeader string, key []byte, body []byte) error {
	sigStr, ok := strings.CutPrefix(sigHeader, "sha1=")
	if !ok {
		return errMissing
	}
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		return errInvalid
	}
	mac := hmac.New(sha1.New, key)
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return errWrong
	}
	return nil
}

func (gh *GitHubWebhook) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	event := req.Header.Get("X-GitHub-Event")
	if event != "push" {
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("io.ReadAll failed: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if gh.Secret != "" {
		sigStr := req.Header.Get("X-Hub-Signature")
		err := validateSignature(sigStr, []byte(gh.Secret), body)
		if err != nil {
			log.Printf("Validate signature failed: %s\n", err)
			http.Error(w, err.Error()+"\n", http.StatusForbidden)
			return
		}
	}

	var payload GitPullPayload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if payload.Ref != "refs/heads/"+gh.Branch {
		log.Printf("Ignoring ref %q\n", payload.Ref)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("Received push to %q\n", payload.Ref)
	cmd := exec.Command("/bin/sh", "-c", "git fetch origin "+gh.Branch+" && git reset --hard FETCH_HEAD")
	cmd.Dir = gh.Path
	err = cmd.Run()
	if err != nil {
		log.Printf("`git pull` failed: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func NewGitHubWebhook(config json.RawMessage) (common.Service, error) {
	gh := new(GitHubWebhook)
	err := json.Unmarshal(config, gh)
	if err != nil {
		return nil, err
	}
	return gh, nil
}

func init() {
	common.Services.Register("github.webhook", NewGitHubWebhook)
}
