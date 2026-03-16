# ted Feature Roadmap Design

## Overview

ted는 Go 기반 개인용 터미널 에디터로, 실사용을 목표로 한다. 현재 piece table 버퍼, tree-sitter 하이라이팅, LSP, git 연동, 검색 등 핵심 기능이 구현되어 있다 (13,500줄, 65개 Go 파일).

이 문서는 다음 단계 기능 로드맵을 정의한다. 전략은 **선택적 참고** — fresh(Rust 터미널 에디터)와 serie(Rust git graph TUI)의 설계와 UX를 참고하되, ted의 Go 코드베이스에 맞게 자체 설계로 구현한다.

## Approach: Selective Reference

- 오픈소스 코드를 읽고 "어떻게 풀었는지" 이해한 뒤, ted 아키텍처에 맞게 재설계
- 코드를 직접 포팅하거나 포크하지 않음
- Rust -> Go 번역이 아닌, 아이디어/패턴 차용

### Reference Projects

- **fresh** (sinelaw/fresh): Rust 터미널 에디터. command palette, split pane, 대용량 파일, 플러그인(QuickJS) 참고
- **serie** (lusingander/serie): Rust git graph TUI. git graph 시각화 알고리즘 참고

## Phase 1: Command Palette / Fuzzy Finder

**참고 소스**: fresh의 command palette 구현

### Goals

- 파일 열기, 명령 실행, goto line, 버퍼 전환을 하나의 통합 인터페이스로 제공
- fuzzy matching 기반 실시간 필터링

### Design

- 오버레이 UI 컴포넌트로 구현 (에디터 위에 떠서 동작)
- 카테고리 분리: files (`Ctrl+P`), commands (`Ctrl+Shift+P`), goto line (`Ctrl+G`)
- fuzzy matching: Go 라이브러리 활용 (e.g. `sahilm/fuzzy`) 또는 fzf 알고리즘 직접 구현
- 키보드 네비게이션: 위/아래 화살표, Enter 선택, Esc 닫기

### fresh 참고 포인트

- palette 진입/퇴장 흐름 (모달 오버레이 패턴)
- 카테고리 prefix 방식 (`>` for commands, `#` for symbols 등)
- 결과 하이라이팅 (매칭된 글자 강조)

## Phase 2: Split Pane

**참고 소스**: fresh의 split view 구조

### Goals

- 수평/수직 분할로 여러 파일을 동시에 볼 수 있게 함
- 같은 파일을 두 pane에서 열기 지원

### Design

- 레이아웃 매니저 도입: 현재 단일 에디터 뷰 -> 트리 구조 레이아웃
- 각 pane은 독립적인 에디터 인스턴스, 버퍼는 공유 가능
- 포커스 관리: `Ctrl+\` 수직 분할, `Ctrl+-` 수평 분할, `Ctrl+W` + 방향키 포커스 전환

### fresh 참고 포인트

- pane 간 리사이즈 처리
- 버퍼 공유 시 커서/스크롤 독립 유지 방식
- pane 닫기/재배치 로직

### 구현 시 고려사항

- 기존 `internal/view/` 구조를 레이아웃 매니저로 확장
- 렌더링 루프에서 각 pane의 visible area를 독립적으로 계산

## Phase 3: Large File Handling

**참고 소스**: fresh의 huge file 블로그 포스트 및 구현

### Goals

- 수백MB급 파일을 낮은 메모리 사용량으로 열고 편집
- 렌더링 지연 없이 스크롤 가능

### Design

- **Lazy loading**: 파일 전체를 메모리에 올리지 않고, 필요한 chunk만 읽기
- **mmap 활용**: `mmap-go` 등으로 OS 가상 메모리에 매핑
- **Visible range 렌더링**: 화면에 보이는 줄만 syntax highlight 및 렌더링
- piece table이 원본 데이터를 복사하지 않고 참조하도록 개선

### Go 특화 고려사항

- GC 부담 최소화: 대용량 처리 시 할당 줄이기 (sync.Pool, 버퍼 재사용)
- mmap과 GC 상호작용 주의: mmap된 메모리는 GC가 관리하지 않으므로 명시적 해제 필요
- 벤치마크 기반 최적화: 100MB, 500MB, 1GB 파일로 메모리/latency 측정

## Phase 4: Git Graph

**참고 소스**: serie 전체

### Goals

- git log --graph를 리치한 TUI로 시각화
- ted 내 통합 뷰로 제공 (별도 탭 또는 패널)

### Design

- commit DAG -> 시각적 그래프 변환 알고리즘 구현
- 브랜치별 컬러링, merge 라인 렌더링
- 스크롤, 커밋 상세 보기, diff 연동

### serie 참고 포인트

- DAG 레이아웃 알고리즘 (commit 위치 배정, 라인 경로 계산)
- 브랜치 색상 할당 전략
- 커밋 메시지/작성자/날짜 표시 레이아웃

### 기존 ted 연동

- `internal/git/` 패키지에 graph 기능 추가
- 이미 있는 blame, diff, stage/commit/push와 자연스럽게 통합

## Phase 5: Plugin System

**참고 소스**: fresh의 TypeScript 플러그인 (QuickJS 샌드박스)

### Goals

- 사용자가 ted의 기능을 확장할 수 있는 플러그인 API 제공
- 안전한 샌드박스 실행

### Design Options (추후 결정)

1. **Lua** (neovim 스타일): Go에서 Lua 임베딩이 성숙 (gopher-lua), 가벼움
2. **JavaScript** (fresh 스타일): QuickJS 바인딩 활용, 웹 개발자 친화적
3. **Go native plugin**: `plugin` 패키지 또는 RPC 기반

### 선결 조건

- Phase 1~4를 통해 내부 API surface가 안정화되어야 함
- 플러그인에 노출할 API 경계를 명확히 정의

## Success Criteria

- Phase 1 완료 후 ted를 일상 에디터로 전환하여 사용 시작
- 각 Phase 완료 시 해당 기능에 대한 테스트 커버리지 확보
- 대용량 파일: 100MB 파일 열기 < 1초, 메모리 사용량 < 파일 크기의 2배
