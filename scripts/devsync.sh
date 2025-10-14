#!/usr/bin/env bash
set -euo pipefail

# --- helpers ---
default_branch() {
  local current
  current=$(git branch --show-current 2>/dev/null || true)
  if [[ -n "$current" ]]; then
    printf '%s' "$current"
    return 0
  fi

  local commit
  commit=$(git rev-parse HEAD 2>/dev/null || true)
  if [[ -n "$commit" ]]; then
    printf '%s' "$commit"
    return 0
  fi

  printf '%s' "main"
}

# --- config ---
BRANCH="${BRANCH:-$(default_branch)}"
BIN_DIR="${BIN_DIR:-bin}"
APP_NAME="${APP_NAME:-ca}"
BIN_PATH="${BIN_PATH:-$BIN_DIR/$APP_NAME}"
PID_FILE="${PID_FILE:-$BIN_DIR/$APP_NAME.pid}"
PORT_FILE="${PORT_FILE:-$BIN_DIR/port}"
BUILD_CMD="${BUILD_CMD:-go build -tags ebiten -o ${BIN_PATH}.new ./cmd/ca}"
RUN_CMD="${RUN_CMD:-./${BIN_PATH}}"
POLL_SECONDS="${POLL_SECONDS:-10}"
PORT="${PORT:-8080}"
AUTO_BUMP="${AUTO_BUMP:-0}"
PORT_GUARD="${PORT_GUARD:-0}"

mkdir -p "$(dirname "$BIN_PATH")"
SRV_PID=""
RUN_ARGS=("$@")

port_in_use() {
  local p="${1:?port required}"
  lsof -iTCP:"$p" -sTCP:LISTEN -t >/dev/null 2>&1
}

wait_port_free() {
  local p="${1:?port required}"
  for _ in {1..30}; do port_in_use "$p" || return 0; sleep 0.1; done
  return 1
}

kill_pid_if_running() {
  local pid="${1:-}"
  [[ -z "$pid" ]] && return 0
  if ps -p "$pid" >/dev/null 2>&1; then
    kill "$pid" 2>/dev/null || true
    for _ in {1..20}; do ps -p "$pid" >/dev/null 2>&1 || break; sleep 0.1; done
    ps -p "$pid" >/dev/null 2>&1 && kill -9 "$pid" 2>/dev/null || true
  fi
}

ensure_port_clear() {
  if [[ -f "$PID_FILE" ]]; then
    local oldpid
    oldpid="$(cat "$PID_FILE" || true)"
    kill_pid_if_running "$oldpid"
    rm -f "$PID_FILE" || true
  fi

  if port_in_use "$PORT"; then
    lsof -iTCP:"$PORT" -sTCP:LISTEN -t 2>/dev/null | xargs -I{} bash -c 'kill {} 2>/dev/null || true'
    sleep 0.2
    if port_in_use "$PORT"; then
      lsof -iTCP:"$PORT" -sTCP:LISTEN -t 2>/dev/null | xargs -I{} bash -c 'kill -9 {} 2>/dev/null || true'
    fi
  fi

  wait_port_free "$PORT"
}

pick_port_if_needed() {
  if [[ "$PORT_GUARD" != "1" ]]; then
    return 0
  fi
  if [[ "$AUTO_BUMP" != "1" ]]; then
    ensure_port_clear || { echo "❌ Port $PORT is busy; set AUTO_BUMP=1 to auto-pick."; exit 1; }
    echo "$PORT" > "$PORT_FILE"
    return 0
  fi

  local base="$PORT"
  for p in "$base" $((base+1)) $((base+2)) $((base+3)) $((base+4)) $((base+5)); do
    PORT="$p"
    if ensure_port_clear; then
      echo "🔌 using port $PORT"
      echo "$PORT" > "$PORT_FILE"
      return 0
    fi
  done
  echo "❌ No free port in range $base..$((base+5))"; exit 1
}

start_server() {
  pick_port_if_needed
  if [[ "$RUN_CMD" == "./${BIN_PATH}" ]]; then
    PORT="$PORT" $RUN_CMD "${RUN_ARGS[@]}" &
  else
    $RUN_CMD "${RUN_ARGS[@]}" &
  fi
  SRV_PID=$!
  echo "$SRV_PID" > "$PID_FILE"
  echo "▶️  started pid=$SRV_PID @ $(date)"
}

stop_server() {
  if [[ -n "${SRV_PID:-}" ]] && ps -p "$SRV_PID" >/dev/null 2>&1; then
    kill "$SRV_PID" 2>/dev/null || true
    wait "$SRV_PID" 2>/dev/null || true
    echo "⏹  stopped pid=$SRV_PID"
  fi
  SRV_PID=""
}

build_swap_run() {
  echo "🔨 building…"
  if $BUILD_CMD; then
    mv "${BIN_PATH}.new" "$BIN_PATH"
    echo "✅ build ok"
    stop_server
    if [[ "$PORT_GUARD" == "1" ]]; then
      wait_port_free "$PORT" || true
    fi
    start_server
  else
    echo "❌ build failed; keeping old binary running"
    rm -f "${BIN_PATH}.new" || true
  fi
}

cleanup() {
  stop_server
  rm -f "$PID_FILE" || true
  echo "🧹 cleaned up"
}
trap cleanup INT TERM EXIT

# --- initial sync & start ---
git fetch origin

FOLLOW_REF=""
if git show-ref --verify --quiet "refs/remotes/origin/$BRANCH"; then
  FOLLOW_REF="origin/$BRANCH"
fi

if git show-ref --verify --quiet "refs/heads/$BRANCH"; then
  git switch -f "$BRANCH"
elif [[ -n "$FOLLOW_REF" ]]; then
  git switch -c "$BRANCH" --track "$FOLLOW_REF"
else
  git switch --detach "$BRANCH"
fi

if [[ -n "$FOLLOW_REF" ]]; then
  git reset --hard "$FOLLOW_REF"
else
  git reset --hard "$BRANCH"
fi
build_swap_run

while sleep "$POLL_SECONDS"; do
  git fetch origin --quiet

  UPDATE_REF="origin/$BRANCH"
  if ! git show-ref --verify --quiet "refs/remotes/$UPDATE_REF"; then
    continue
  fi

  LOCAL=$(git rev-parse HEAD)
  REMOTE=$(git rev-parse "$UPDATE_REF")
  if [[ "$LOCAL" != "$REMOTE" ]]; then
    echo "⬇️  upstream changed -> updating to $REMOTE"
    git reset --hard "$UPDATE_REF"
    build_swap_run
  fi
done
