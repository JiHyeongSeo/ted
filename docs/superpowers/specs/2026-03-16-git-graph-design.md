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
- 가로 스크롤 (그래프 열이 터미널 폭을 넘으면 표시 가능한 만큼만 렌더링, 나머지 truncate)
- 상하 분할 비율 조절 (v1에서는 70/30 고정)

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
type GraphCell int

const (
    CellEmpty    GraphCell = iota // 빈칸
    CellCommit                    // ● 커밋 노드
    CellPipe                      // │ 수직 연속선
    CellMergeRight                // ┘ 오른쪽에서 왼쪽으로 병합
    CellBranchRight               // ┐ 왼쪽에서 오른쪽으로 분기
    CellMergeLeft                 // └ 왼쪽에서 오른쪽으로 병합
    CellBranchLeft                // ┌ 오른쪽에서 왼쪽으로 분기
    CellHorizontal                // ─ 수평 연결선
    CellCross                     // ┼ 교차
)

type GraphRow struct {
    Commit   *Commit
    Cells    []GraphCell // 그래프 열 데이터 (columns 수만큼)
    Colors   []int       // 각 열의 색상 인덱스
    Column   int         // 이 커밋의 열 위치
}

func LayoutGraph(commits []Commit) []GraphRow
```

**레이아웃 알고리즘 (serie 참고):**

`activeLanes []string` — 현재 활성 열 목록. 각 열은 추적 중인 커밋 해시(다음에 나타날 부모)를 저장.

커밋을 위에서 아래로 순회하며:

1. **열 찾기**: 현재 커밋 해시가 activeLanes의 어느 열에 있는지 찾음. 없으면 빈 열(또는 끝에 추가)에 배치.
2. **부모 연결**:
   - 부모가 1개 (일반 커밋): 해당 열에 부모 해시를 넣어 계속 추적
   - 부모가 2개 (merge 커밋): 첫 번째 부모는 같은 열에서 계속, 두 번째 부모는 해당 열이 있으면 그쪽으로 병합선(`┘`), 없으면 새 열에 분기선(`┐`)
   - 부모가 0개 (root 커밋): 해당 열 해제
3. **열 해제**: 더 이상 추적할 부모가 없는 열은 제거. 빈 열은 재사용 가능.
4. **색상**: 열이 처음 할당될 때 색상 인덱스 배정 (열 인덱스 % 팔레트 크기). 열이 살아있는 동안 색상 유지.

**렌더링 예시:**

```
Linear:          Branch + Merge:      Two branches:
● commit A       ● commit A           ● commit A
│                ├─┐                  ├─┐
● commit B       │ ● commit B         ● │ commit B
│                │ │                  │ ● commit C
● commit C       ● │ commit C         ├─┘
                 ├─┘                  ● commit D (merge)
                 ● commit D (merge)
```

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

- 선택된 커밋 정보 표시: 전체 해시, 커밋 메시지 (첫 줄), 작성자, 날짜
- 변경 파일 목록: `git diff-tree --no-commit-id --name-status -r <hash>` 로 로드
- 파일당 한 줄씩 표시 (상태 아이콘 + 파일 경로): `M internal/buffer/buffer.go`
- 스크롤 지원 (파일 목록이 뷰 높이를 넘는 경우)

### 5. `internal/editor/editor_graph.go` — 에디터 통합

- `git.graph` 명령어 등록 (커맨드 팔레트에서 실행 가능)
- `Ctrl+Shift+G` 키바인딩
- GraphView + CommitDetailView 를 상하 분할로 배치
- 탭 시스템에 특수 탭 타입으로 등록

## 탭 시스템 확장

현재 `Tab` 구조체는 `Buffer` 기반이다. 그래프 탭을 위해 확장:

```go
type TabKind int
const (
    TabKindFile  TabKind = iota // 기존 파일 편집 탭
    TabKindGraph                // git graph 탭
)

type Tab struct {
    Kind     TabKind
    Buffer   *buffer.Buffer // TabKindFile일 때 사용
    Language string
    // Graph 관련 필드는 editor_graph.go에서 별도 관리
}
```

- 그래프 탭은 **싱글 인스턴스** — 이미 열려 있으면 해당 탭으로 포커스 이동
- 그래프 탭 닫기: `buf.Close()` 대신 GraphView 리소스 해제
- 그래프 탭에서 Enter → `git show <hash>` 출력을 DiffView로 새 탭에 열기

## 기존 코드 연동

- `DiffTracker`에 `LoadCommits` 메서드 추가
- 렌더링: `TabKindGraph`일 때 `EditorView` 대신 `GraphView` + `CommitDetailView` 렌더링
- 테마: 기존 `syntax.Theme`의 UI 스타일 활용, 브랜치 색상은 별도 팔레트

## 에러 처리

- git 저장소가 아닌 경우: 상태바에 "Not a git repository" 표시, 탭 열지 않음
- 커밋이 0개인 경우: 그래프 영역에 "No commits yet" 메시지 표시
- `git log` 실패: 상태바에 에러 메시지 표시
- 터미널 폭이 최소 너비(40) 미만: 그래프 열 생략, 메시지만 표시

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

열이 처음 할당될 때 색상 배정. 열이 해제되고 재사용되어도 새 색상 배정.

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
