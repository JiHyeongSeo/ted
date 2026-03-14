# Claude Integration — Design Spec

## Overview

ted 에디터에 Claude Code CLI를 subprocess로 연동하여 인라인 코드 편집과 채팅 기능을 제공한다.

## Decisions

| 항목 | 결정 |
|------|------|
| AI 백엔드 | Claude Code CLI (`claude`) subprocess |
| 호출 방식 | `claude -p "prompt" --output-format stream-json` |
| 채팅 맥락 | `--continue` 플래그로 세션 유지 |
| 파일 수정 | ted가 응답에서 코드 추출 → 버퍼에 적용 (undo 지원) |
| 인라인 트리거 | `Ctrl+K` — 미니 입력창 |
| 채팅 트리거 | `Ctrl+L` — 채팅 패널 |

## Architecture

```
Editor (orchestrator)
  ├── claude.Runner          ← subprocess 관리 + stream-json 파싱
  ├── view.ChatPanel         ← 채팅 패널 (히스토리 + 입력)
  ├── view.InlinePrompt      ← Ctrl+K 입력 오버레이 (InputBar 재활용)
  └── ExecuteCommand
        ├── claude.inline    → 선택 코드 + 지시 → 응답으로 교체
        └── claude.chat      → 채팅 패널 대화
```

## Component: `internal/claude/runner.go`

### Types

```go
package claude

type Runner struct {
    workDir       string
    sessionActive bool
}

type StreamEvent struct {
    Type    string // "text", "done", "error"
    Content string
}

type Option func(*runConfig)
type runConfig struct {
    continue_ bool
}

func WithContinue() Option
```

### Interface

```go
func NewRunner(workDir string) *Runner

// Run spawns `claude -p` and streams events via channel.
// Caller reads events until channel closes.
func (r *Runner) Run(prompt string, opts ...Option) (<-chan StreamEvent, error)

// Cancel kills the running subprocess.
func (r *Runner) Cancel()

// IsRunning returns true if a subprocess is active.
func (r *Runner) IsRunning() bool
```

### Behavior

1. Constructs command: `claude -p "<prompt>" --output-format stream-json [--continue]`
2. Sets working directory to project root
3. Starts process, reads stdout line by line
4. Each line is JSON — parse and emit StreamEvent
5. On process exit, close channel
6. stream-json format: each line is `{"type":"assistant","subtype":"text","text":"..."}` or similar

### stream-json Event Parsing

Claude Code의 `--output-format stream-json` 출력:
- `{"type":"assistant","subtype":"text","text":"chunk"}` — 텍스트 청크
- `{"type":"result","result":"full text","cost":...}` — 최종 결과

파싱 전략: `subtype == "text"` → StreamEvent{Type: "text"}, `type == "result"` → StreamEvent{Type: "done"}

## Component: Inline Edit (`Ctrl+K`)

### Flow

1. 사용자가 코드 선택 (선택 없으면 현재 줄)
2. `Ctrl+K` → InputBar 표시 ("AI: " 프롬프트)
3. 지시 입력 후 Enter
4. 프롬프트 구성:
   ```
   File: {filename}
   Language: {language}

   Selected code:
   ```{lang}
   {selected_code}
   ```

   Instruction: {user_instruction}

   Return ONLY the replacement code, no explanation.
   ```
5. Runner.Run(prompt) 실행
6. 상태바: "Claude thinking..."
7. 응답 수신 완료 → 코드 블록 추출 (```...``` 사이)
8. 선택 영역을 추출된 코드로 교체 (EditorView.ReplaceSelection)
9. 상태바: "Applied" / 에러 시 에러 메시지

### 코드 블록 추출

응답에서 첫 번째 코드 블록 (``` 로 시작하는 부분)을 추출.
코드 블록이 없으면 응답 전체를 코드로 간주.

## Component: Chat Panel (`Ctrl+L`)

### UI Layout

하단 패널의 "Claude" 탭 (기존 Problems/Output/Terminal 옆에 추가):

```
┌─ Problems ─ Output ─ Terminal ─ [Claude] ─────────┐
│ User: 이 함수 설명해줘                              │
│                                                     │
│ Claude: 이 함수는 입력 문자열을 파싱하여...          │
│ ```go                                               │
│ func Parse(s string) Result {                       │
│ ```                                                 │
│ [코드 블록 — Enter로 적용]                           │
│                                                     │
│ > 입력창: _                                         │
└─────────────────────────────────────────────────────┘
```

### 동작

1. `Ctrl+L` → 채팅 패널 열기 + 포커스
2. 선택된 코드가 있으면 자동으로 컨텍스트 첨부
3. 입력 후 Enter → Runner.Run(prompt, WithContinue())
4. 스트리밍 응답을 실시간으로 패널에 표시
5. 코드 블록 감지: 패널에서 코드 블록 줄에 커서 놓고 Enter → 에디터 버퍼에 삽입

### ChatPanel 구조

```go
type ChatPanel struct {
    BaseComponent
    theme     *syntax.Theme
    messages  []ChatMessage
    inputBuf  []rune
    inputCur  int
    scrollY   int
    streaming bool // true while receiving Claude response
}

type ChatMessage struct {
    Role    string // "user", "assistant"
    Content string
}
```

### 키 바인딩

- `Ctrl+L`: 채팅 패널 열기/포커스
- 패널 내 Enter: 메시지 전송 (스트리밍 중이면 무시)
- 패널 내 Escape: 포커스 해제
- 패널 내 Up/Down: 히스토리 스크롤

## Editor Integration

### 새 필드

```go
type Editor struct {
    // ... existing fields ...
    claudeRunner *claude.Runner
    chatPanel    *view.ChatPanel
}
```

### 커맨드 등록

| 커맨드 | 단축키 | 설명 |
|--------|--------|------|
| `claude.inline` | `Ctrl+K` | 인라인 AI 편집 |
| `claude.chat` | `Ctrl+L` | 채팅 패널 열기 |
| `claude.cancel` | — | 진행 중인 요청 취소 |

### EditorView 추가 메서드

```go
// ReplaceSelection replaces selected text with new text (undo-able).
func (e *EditorView) ReplaceSelection(text string)
```

## Error Handling

- `claude` CLI 미설치: 상태바에 "Claude Code not found. Install: npm i -g @anthropic-ai/claude-code"
- 프로세스 실패: 상태바에 에러 메시지
- 스트리밍 중 취소: Runner.Cancel() → 프로세스 kill
- 빈 응답: "No response from Claude"
