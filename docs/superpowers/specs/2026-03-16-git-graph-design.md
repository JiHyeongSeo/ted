# Git Graph Design

## Overview

ted에 git commit history를 시각적 그래프로 보여주는 읽기 전용 뷰를 추가한다. 별도 탭으로 열리며, 상하 분할 레이아웃으로 위쪽은 커밋 그래프, 아래쪽은 선택된 커밋의 상세 정보를 표시한다.

**참고 소스**: serie (lusingander/serie) — Rust git graph TUI의 DAG 레이아웃 알고리즘, 브랜치 색상 전략 참고

## Goals

- `git log --graph`를 리치한 TUI로 시각화
- ted 내 별도 탭으로 제공
- 커밋 선택 시 상세 정보(해시, 작성자, 날짜, 메시지, 변경 파일 목록) 표시
- 읽기 전용 (git 조작 기능 없음)

## Non-Goals

- 브랜치 체크아웃, cherry-pick, rebase 등 git 조작
- refs 필터링, 커밋 검색 등 고급 탐색 기능
- 가로 스크롤 (그래프 열 수가 터미널 폭을 넘는 케이스는 1차에서 미지원)

## Architecture

### 데이터 흐름

```
git log --format → 파싱 → []Commit (DAG) → 레이아웃 알고리즘 → []GraphRow → 렌더링
```

1. `git log`로 커밋 데이터 수집 (해시, 부모, 작성자, 날짜, 메시지, refs)
2. 부모 해시로 DAG(Directed Acyclic Graph) 구성
3. 레이아웃 알고리즘이 각 커밋에 열(column) 배정 + 연결선 경로 계산
4. GraphView가 visible range만 tcell로 렌더링

### UI 레이아웃

```
┌─────────────────────────────────────────────┐
│ [main.go] [buffer.go] [⎇ Git Graph]        │  ← 탭 바
├─────────────────────────────────────────────┤
│ ● ─ bd0abad  test: add benchmarks   seoji 2m│
│ ● ─ 4a9a420  perf: avoid buf.Text() seoji 5m│  ← 그래프 영역 (상단)
│ ● ─ 9308f23  feat: mmap-based file   seoji 8m│     스크롤 가능, 행 선택
│ ● ┬ ce26793  refactor: migrate PT    seoji 12│
│ │ ● f6e98ad  feat: add MmapContent   seoji 15│
│ ● ┘ e0a38f0  feat: add ContentSource seoji 18│
├─────────────────────────────────────────────┤
│ Commit: 9308f23                              │
│ feat: mmap-based file loading (>10MB)        │  ← 커밋 상세 (하단)
│ Author: seoji · Date: 2026-03-16 14:30       │     선택된 커밋의 정보
│ Files: M buffer.go, A content.go             │
└─────────────────────────────────────────────┘
```

- 상하 비율: 그래프 70%, 상세 30% (고정)
- 그래프 영역: 스크롤, 행 선택 (↑↓, 마우스 클릭)
- 상세 영역: 선택된 커밋의 해시, 메시지, 작성자, 날짜, 변경 파일 목록

## Components

### 1. `internal/git/graph.go` — 데이터 모델 + git log 파싱

```go
type Commit struct {
    Hash      string
    ShortHash string
    Parents   []string
    Author    string
    Date      time.Time
    Message   string   // 첫 줄만
    Refs      []string // branch/tag names
}

func LoadCommits(repoRoot string, maxCount int) ([]Commit, error)
```

- `git log --format=...` 으로 커밋 데이터 파싱
- `--max-count` 로 로딩 제한 (기본 500개)
- 부모 해시 포함하여 DAG 관계 추출

### 2. `internal/git/graph_layout.go` — DAG 레이아웃 알고리즘

```go
type GraphCell int // 빈칸, 커밋노드, 수직선, 분기, 병합 등

type GraphRow struct {
    Commit   *Commit
    Cells    []GraphCell // 그래프 열 데이터
    Column   int         // 이 커밋의 열 위치
    Color    int         // 브랜치 색상 인덱스
}

func LayoutGraph(commits []Commit) []GraphRow
```

**레이아웃 알고리즘 (serie 참고):**
- 각 활성 브랜치에 열(column) 할당
- 커밋을 위에서 아래로 순회하며:
  - 새 브랜치 시작 → 빈 열 할당
  - 병합(merge) → 자식 열에서 부모 열로 연결선
  - 브랜치 종료 → 열 해제 (재사용 가능)
- 브랜치별 색상: 열 인덱스 기반으로 사전 정의된 색상 팔레트에서 순환 배정

### 3. `internal/view/graphview.go` — 그래프 렌더링

```go
type GraphView struct {
    BaseComponent
    rows        []GraphRow
    selectedIdx int
    scrollY     int
    onSelect    func(commit *Commit)
}
```

- tcell 기반 렌더링
- box drawing 문자: `●` (커밋), `│` (수직선), `─` (수평선), `┬` `┘` `├` (분기/병합)
- 각 행: `[그래프] [해시] [메시지] [작성자] [날짜]` 열 정렬
- visible range만 렌더링 (scrollY 기반)
- 키보드: ↑↓ 행 이동, PgUp/PgDn, Home/End
- 마우스: 클릭 선택, 휠 스크롤

### 4. `internal/view/commitdetail.go` — 커밋 상세 뷰

```go
type CommitDetailView struct {
    BaseComponent
    commit   *Commit
    files    []string // 변경 파일 목록
    scrollY  int
}
```

- 선택된 커밋의 전체 정보 표시
- `git diff-tree --no-commit-id --name-status -r <hash>` 로 변경 파일 목록 로드
- 스크롤 지원 (파일 목록이 길 경우)

### 5. `internal/editor/editor_graph.go` — 에디터 통합

- `git.graph` 명령어 등록 (커맨드 팔레트에서 실행 가능)
- `Ctrl+Shift+G` 키바인딩 (또는 설정 가능)
- GraphView + CommitDetailView 를 상하 분할로 배치
- 탭 시스템에 특수 탭 타입으로 등록

## 기존 코드 연동

- `DiffTracker`에 `LoadCommits` 메서드 추가 (또는 별도 함수)
- 탭 시스템: 현재 `Tab`은 `Buffer` 기반 — GraphView용 특수 탭 타입 필요
- 렌더링: 기존 `EditorView` 자리에 `GraphView` + `CommitDetailView` 표시
- 테마: 기존 `syntax.Theme`의 UI 스타일 활용, 브랜치 색상은 별도 팔레트

## 브랜치 색상 팔레트

serie 스타일로 고정 색상 8개 순환:

```go
var BranchColors = []tcell.Color{
    tcell.ColorRed,
    tcell.ColorGreen,
    tcell.ColorYellow,
    tcell.ColorBlue,
    tcell.ColorDarkCyan,
    tcell.ColorFuchsia,
    tcell.ColorOrange,
    tcell.ColorLightGray,
}
```

열 인덱스 % len(BranchColors) 로 배정.

## 키바인딩

| 키 | 동작 |
|----|------|
| ↑/↓ | 커밋 선택 이동 |
| PgUp/PgDn | 페이지 스크롤 |
| Home/End | 처음/끝으로 |
| Enter | 선택된 커밋의 diff를 DiffView 탭으로 열기 |
| Esc / Ctrl+W | 그래프 탭 닫기 |
| 마우스 클릭 | 커밋 선택 |
| 마우스 휠 | 스크롤 |

## 테스트 전략

- `graph.go`: git log 파싱 단위 테스트 (고정 출력 문자열 파싱)
- `graph_layout.go`: 레이아웃 알고리즘 단위 테스트 (선형, 분기, 병합, 복잡 DAG)
- `graphview.go`: 뷰 상태 테스트 (스크롤, 선택, 경계 조건)
- `commitdetail.go`: 파일 목록 파싱 테스트
- 통합: 실제 git repo에서 그래프 열기 → 커밋 수 확인
