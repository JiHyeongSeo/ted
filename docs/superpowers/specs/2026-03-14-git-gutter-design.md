# Git Gutter Diff Markers — Design Spec

## Overview

EditorView의 라인 번호 영역에 git diff 상태(added/modified/deleted)를 배경색으로 표시하는 기능.

## Decisions

| 항목 | 결정 |
|------|------|
| 시각 스타일 | Block Style — 라인 번호 칸 전체 배경색 |
| 삭제 표시 | 삭제 발생 직후 라인에 빨간 배경 |
| 데이터 소스 | `git diff` CLI (`git diff --unified=0 HEAD -- <file>`) |
| 갱신 타이밍 | 파일 저장 시 + 파일 열기/탭 전환 시 |
| 테마 연동 | 기존 UI 맵에 `gitAdded`, `gitModified`, `gitDeleted` 키 추가 |
| 구현 방식 | EditorView 내장 (별도 컴포넌트 없음) |

## Architecture

```
Editor (orchestrator)
  ├── git.DiffTracker        ← NEW
  ├── EditorView
  │     └── gutterMarkers    ← NEW field
  └── Theme.UI
        ├── gitAdded         ← NEW key
        ├── gitModified      ← NEW key
        └── gitDeleted       ← NEW key
```

## Component: `internal/types/gutter.go`

### GutterMark Type (shared)

`GutterMark` 타입은 `internal/types`에 정의하여 `git`과 `view` 패키지 간 순환 의존 방지:

```go
package types

type GutterMark int

const (
    MarkNone     GutterMark = iota
    MarkAdded
    MarkModified
    MarkDeleted
)
```

## Component: `internal/git/diff.go`

### Types

```go
package git

type DiffTracker struct {
    repoRoot string
}
```

### Constructor

```go
// NewDiffTracker: git rev-parse --show-toplevel로 repo root 탐색.
// git repo가 아니면 nil, nil 반환 (에러 아닌 정상 케이스).
func NewDiffTracker(projectRoot string) (*DiffTracker, error)
```

### Core Method

```go
// ComputeMarkers: 파일의 git diff를 계산해 라인별 마커 맵 반환.
// 0-based 라인 인덱스 → GutterMark.
func (dt *DiffTracker) ComputeMarkers(filePath string) (map[int]GutterMark, error)
```

### Parsing Logic

`git diff --unified=0 HEAD -- <relPath>` 실행 후 `@@` hunk 헤더 파싱:

```
@@ -oldStart,oldCount +newStart,newCount @@
```

**주의: git diff 출력의 라인 번호는 1-based. 마커 맵은 0-based. 변환 필요.**

- `oldCount == 0` → 순수 추가: `(newStart-1)` ~ `(newStart-1+newCount-1)` 라인에 `MarkAdded`
- `newCount == 0` → 순수 삭제: `max(0, newStart-1)` 라인에 `MarkDeleted`
  - `newStart`가 0인 경우 (파일 맨 앞에서 삭제): 라인 0에 표시
- 그 외 → 수정: `(newStart-1)` ~ `(newStart-1+newCount-1)` 라인에 `MarkModified`

### Untracked 파일 감지

`git diff`가 빈 출력을 반환하면 `git ls-files <file>`로 추적 여부 확인.
추적되지 않은 파일이면 모든 라인을 `MarkAdded`로 표시.
(diff가 비어있을 때만 ls-files를 호출하여 불필요한 서브프로세스 최소화)

### Edge Cases

- **git repo가 아닌 경우**: DiffTracker가 nil → 호출 스킵, 빈 마커
- **새 파일 (미저장, 경로 없음)**: `Buffer.Path() == ""` → 스킵, 빈 마커
- **바이너리 파일**: git diff가 "Binary files differ" 출력 → 빈 맵 반환
- **빈 diff (변경 없음)**: 빈 맵 반환
- **hunk 카운트 생략**: `@@ -10 +10 @@` 형태 (count=1 생략) → 1로 처리
- **파일 맨 앞 삭제**: `newStart == 0` → 라인 0에 MarkDeleted

## Component: EditorView 변경

### 새 필드

```go
gutterMarkers map[int]GutterMark  // 0-based 라인 → 마커 종류
```

### 새 메서드

```go
func (ev *EditorView) SetGutterMarkers(markers map[int]GutterMark)
```

### Render 변경

라인 번호 렌더링 시 해당 라인의 gutterMarker를 확인하여 배경색 오버라이드:

```
for each visible line:
    marker := ev.gutterMarkers[lineIdx]
    style := default lineNumber style (or lineNumberActive if current line)
    if marker != MarkNone:
        // 기존 foreground 유지, 배경색만 교체
        bgColor := theme.ResolveColor(theme.UI["gitAdded"|"gitModified"|"gitDeleted"])
        style = style.Background(bgColor)
    draw line number with style
```

gutter marker 배경은 `lineNumberActive`의 배경보다 우선하되, **foreground(숫자 색상)는 기존 스타일 유지**.
즉 활성 라인의 밝은 라인번호 + git 배경색 조합 가능.

## Component: Editor 연결

### 새 필드

```go
diffTracker *git.DiffTracker  // nil if not a git repo
```

### 초기화

Editor 생성 시:
```go
dt, _ := git.NewDiffTracker(projectRoot)
e.diffTracker = dt  // nil이면 git 기능 비활성
```

### 갱신 포인트

1. **파일 저장 시** (`file.save` 커맨드):
   ```
   buffer.Save()
   → updateGutterMarkers()
   → render()
   ```

2. **파일 열기 / 탭 전환 시** (`syncViewToTab()`):
   ```
   syncViewToTab()
   → updateGutterMarkers()
   ```

### Helper

```go
func (e *Editor) updateGutterMarkers() {
    if e.diffTracker == nil { return }
    tab := e.tabs.Active()
    if tab == nil || tab.Buffer.Path() == "" { return }
    markers, err := e.diffTracker.ComputeMarkers(tab.Buffer.Path())
    if err != nil { return }  // 실패 시 조용히 무시
    e.editorView.SetGutterMarkers(markers)
}
```

## Theme Changes

기존 Theme.UI 맵에 3개 키 추가:

| Key | Default Color | Description |
|-----|---------------|-------------|
| `gitAdded` | `#2d4a2d` | 추가된 라인 — 어두운 녹색 배경 |
| `gitModified` | `#0c3d4d` | 수정된 라인 — 어두운 파란색 배경 |
| `gitDeleted` | `#4d1a1a` | 삭제 위치 — 어두운 빨간색 배경 |

색상은 터미널 배경(#1e1e1e)에서 잘 보이면서 과하지 않은 어두운 톤.
`DefaultTheme()`에 기본값 추가. JSON 테마 파일에 키가 없으면 이 기본값으로 폴백.

## Design Note: `git diff HEAD` vs `git diff`

`git diff HEAD`는 working tree vs HEAD 커밋을 비교한다. 즉 staged 상태를 무시하고 HEAD 대비 전체 변경을 보여준다. VSCode는 기본적으로 `git diff` (working tree vs index)를 사용하지만, 터미널 에디터에서는 staging area를 직접 관리하지 않으므로 HEAD 비교가 더 직관적이다.

## Files Changed

| File | Change |
|------|--------|
| `internal/types/gutter.go` | **NEW** — GutterMark 타입 (순환 의존 방지를 위해 types에 배치) |
| `internal/git/diff.go` | **NEW** — DiffTracker, ComputeMarkers |
| `internal/view/editorview.go` | gutterMarkers 필드, SetGutterMarkers(), Render 수정 |
| `internal/editor/editor.go` | diffTracker 필드, 초기화, updateGutterMarkers(), 저장/열기 시 호출 |
| `internal/syntax/theme.go` | gitAdded/gitModified/gitDeleted 기본값 추가 + 폴백 처리 |

## Not In Scope

- Git blame
- Git commit/push/pull UI
- Inline diff 보기 (gutter 클릭 시 diff 상세)
- 실시간 갱신 (타이핑 중)
- 설정으로 gutter 끄기/켜기
