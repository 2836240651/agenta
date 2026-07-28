# Hyhacct Responses Protocol Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Commander use `/v1/responses` for Hyhacct chat and vision analysis while keeping image generation endpoints unchanged.

**Architecture:** Add a protocol selector to `ServerConfigHyhacct`. `HyhacctUtils.ChatCompletion` dispatches either to the existing Chat Completions adapter or a new Responses adapter that converts messages to `input` content and extracts text from `output[].content[].text`. Image APIs remain on the current Resty image client and credentials.

**Tech Stack:** Go, Gin, Resty, `github.com/sashabaranov/go-openai`, Go `httptest`, YAML configuration, Docker deployment.

## Global Constraints

- Only `api_key_chat` traffic may use `/v1/responses`.
- `/images/generations` and `/images/edits` retain their path, API key, model and multipart behavior.
- `chat_protocol` defaults to `chat_completions` for backward compatibility.
- Never commit, print or document API keys.
- Deploy only after unit tests and a production visual probe pass.

---

### Task 1: Add configuration and Responses tests

**Files:**
- Modify: `commander-server-t260220-main/commander-server-t260220-main/internal/types/server.go`
- Modify: `commander-server-t260220-main/commander-server-t260220-main/internal/utils/hyhacct_test.go`

**Interfaces:**
- Produces: `ServerConfigHyhacct.ChatProtocol string` using YAML key `chat_protocol`.
- Produces: tests requiring `ChatCompletion` to call `/v1/responses` and parse Responses output when the protocol is `responses`.

- [ ] **Step 1: Write failing tests**

```go
func TestHyhacctUtils_ChatCompletionResponsesText(t *testing.T) {
    // Mock asserts POST /v1/responses and input_text payload.
    // Mock returns output[0].content[0].text; test expects that text.
}

func TestHyhacctUtils_ChatWithVisionResponsesUsesInputImage(t *testing.T) {
    // Mock asserts input_image contains the supplied data URL.
}
```

- [ ] **Step 2: Run failing tests**

Run: `go test ./internal/utils -run 'Responses' -count=1`

Expected: FAIL because `chat_protocol` and Responses adapter do not exist.

### Task 2: Implement protocol adapter

**Files:**
- Modify: `commander-server-t260220-main/commander-server-t260220-main/internal/types/server.go`
- Modify: `commander-server-t260220-main/commander-server-t260220-main/internal/utils/hyhacct.go`

**Interfaces:**
- Consumes: `ServerConfigHyhacct.ChatProtocol`.
- Produces: `ChatCompletion(req openai.ChatCompletionRequest) (string, error)` with dual protocol support.

- [ ] **Step 1: Add protocol field and default resolver**

```go
func hyhacctChatProtocol(h types.ServerConfigHyhacct) string {
    if strings.EqualFold(strings.TrimSpace(h.ChatProtocol), "responses") {
        return "responses"
    }
    return "chat_completions"
}
```

- [ ] **Step 2: Add Responses request/response structs and mapper**

```go
type hyhacctResponsesRequest struct {
    Model string `json:"model"`
    Input []hyhacctResponsesInputMessage `json:"input"`
}

type hyhacctResponsesContent struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    ImageURL string `json:"image_url,omitempty"`
}
```

- [ ] **Step 3: Route chat calls and parse output text**

Run: `go test ./internal/utils -run 'ChatCompletion|Responses' -count=1`

Expected: PASS.

### Task 3: Preserve legacy and image behavior

**Files:**
- Modify: `commander-server-t260220-main/commander-server-t260220-main/internal/utils/hyhacct_test.go`

**Interfaces:**
- Consumes: default empty protocol and `responses` protocol.
- Produces: proof that legacy requests still use `/v1/chat/completions` and image edit requests still use `/v1/images/edits`.

- [ ] **Step 1: Add/retain legacy route assertion**
- [ ] **Step 2: Run focused utility suite**

Run: `go test ./internal/utils -count=1`

Expected: PASS.

### Task 4: Build, deploy, and verify

**Files:**
- Modify: production `/data/commander/config/config.yaml` only during user-authorized deployment.

- [ ] **Step 1: Build Linux binary**

Run: `go build -o .local/bin/commander-server-linux-amd64 .`

- [ ] **Step 2: Update production non-secret protocol setting and supplied chat credential**

Set `hyhacct.chat_protocol: responses`; do not alter image settings.

- [ ] **Step 3: Deploy through the existing restore-and-deploy script**

Run: `node scripts/_ssh-probe/restore-config-and-deploy.js`

- [ ] **Step 4: Verify production**

Run a real `/v1/responses` visual probe using the configured chat key, then check Commander container health and recent logs.

Expected: visual probe returns a non-error text completion; `commander-server-t260220` is running.