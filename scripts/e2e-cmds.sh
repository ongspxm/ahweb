#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
binary="$tmp/ahweb"
output="$tmp/output"
installed=0
use_sudo=0

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  if [[ -f "$output" ]]; then cat "$output" >&2; fi
  exit 1
}

service_cmd() {
  if (( use_sudo )); then sudo "$@"; else "$@"; fi
}

run() {
  "$@" >"$output" 2>&1 || fail "$*"
}

case "$(uname -s)" in
  Darwin)
    service_file=/Library/LaunchDaemons/com.ongspxm.ahweb.plist
    log_dir="$HOME/Library/Logs/ahweb"
    use_sudo=1
    status_pattern='state = running'
    ;;
  Linux)
    service_file="$HOME/.config/systemd/user/ahweb.service"
    status_pattern='Active: active \(running\)'
    command -v systemctl >/dev/null || fail 'systemctl not found'
    systemctl --user show-environment >/dev/null 2>&1 || fail 'systemd user manager is not available'
    ;;
  *)
    fail 'supported systems are Linux and macOS'
    ;;
esac

[[ ! -e "$service_file" ]] || fail "service already exists: $service_file; use a clean disposable machine"

cleanup() {
  result=$?
  trap - EXIT
  if (( installed )); then
    service_cmd "$binary" uninstall >/dev/null 2>&1 || {
      printf 'WARNING: uninstall failed; remove the test service manually\n' >&2
    }
  fi
  rm -rf "$tmp"
  exit "$result"
}
trap cleanup EXIT

cd "$root"
printf 'Building test binary...\n'
go build -o "$binary" . || fail 'go build failed'
installed=1 # uninstall also cleans up a partial install
run service_cmd "$binary" install

check_running() {
  for _ in $(seq 1 30); do
    if service_cmd "$binary" status >"$output" 2>&1 && grep -Eq "$status_pattern" "$output"; then
      return
    fi
    sleep 1
  done
  fail 'service did not reach running state'
}

check_logs() {
  case "$(uname -s)" in
    Darwin)
      find "$log_dir" -type f -mmin -2 -print -quit | grep -q . || fail 'no service log updated in the last 2 minutes'
      ;;
    Linux)
      journalctl --user -u ahweb.service --since '2 minutes ago' -n 1 --no-pager >"$output" 2>&1 || fail 'could not read service logs'
      [[ -s "$output" ]] || fail 'no service log found in the last 2 minutes'
      ;;
  esac
}

check_running
printf 'PASS: install (service is running)\n'
check_logs
printf 'PASS: logs (recent service log found)\n'
run service_cmd "$binary" restart
check_running
check_logs
printf 'PASS: logs (recent service log found)\n'
printf 'PASS: restart (service is running; recent log found)\n'
run service_cmd "$binary" uninstall
installed=0

if service_cmd "$binary" status >"$output" 2>&1; then
  fail 'service is still available after uninstall'
fi
[[ ! -e "$service_file" ]] || fail "service file remains after uninstall: $service_file"
printf 'PASS: uninstall (service removed)\n'
