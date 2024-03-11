package github

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
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

type signatureValidator struct {
	mac hash.Hash
}

func newSignatureValidator(secret []byte) *signatureValidator {
	return &signatureValidator{
		mac: hmac.New(sha1.New, secret),
	}
}

func (v *signatureValidator) Write(p []byte) (n int, err error) {
	return v.mac.Write(p)
}

func (v *signatureValidator) Validate(sigHeader string) error {
	sigStr, ok := strings.CutPrefix(sigHeader, "sha1=")
	if !ok {
		return errMissing
	}
	sig, err := hex.DecodeString(sigStr)
	if err != nil {
		return errInvalid
	}
	if !hmac.Equal(sig, v.mac.Sum(nil)) {
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

	// Stream the request body to avoid io.ReadAll eating all the memory
	var jsonReader io.Reader = req.Body
	var validator *signatureValidator
	if gh.Secret != "" {
		validator = newSignatureValidator([]byte(gh.Secret))
		jsonReader = io.TeeReader(req.Body, validator)
	}

	var payload GitPullPayload
	err := json.NewDecoder(jsonReader).Decode(&payload)
	if err != nil {
		log.Printf("json.Unmarshal failed: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	io.Copy(io.Discard, jsonReader) // so that HMAC receives all the body

	if gh.Secret != "" {
		sigStr := req.Header.Get("X-Hub-Signature")
		err := validator.Validate(sigStr)
		if err != nil {
			log.Printf("Validate signature failed: %s\n", err)
			http.Error(w, err.Error()+"\n", http.StatusForbidden)
			return
		}
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
	common.Services.Register("github.webhook.pull", NewGitHubWebhook)
}
