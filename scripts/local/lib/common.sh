#!/usr/bin/env bash

local_to_lower() {
  printf '%s\n' "$1" | tr '[:upper:]' '[:lower:]'
}

to_lower() {
  local_to_lower "$@"
}

is_truthy() {
  case "$(local_to_lower "${1:-}")" in
    1|true|yes|on)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

require_command() {
  local command_name="$1"
  local usage_context="${2:-this local script}"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "$command_name is required for $usage_context" >&2
    return 1
  fi
}

normalize_http_url() {
  local value="$1"
  if [[ "$value" != http://* && "$value" != https://* ]]; then
    value="http://$value"
  fi
  printf '%s\n' "${value%/}"
}

url_host() {
  local url="$1"
  local rest host_port host
  rest="${url#*://}"
  host_port="${rest%%/*}"
  if [[ "$host_port" == \[*\]* ]]; then
    host="${host_port#\[}"
    host="${host%%\]*}"
  else
    host="${host_port%%:*}"
  fi
  printf '%s\n' "$host"
}

is_loopback_host() {
  case "$1" in
    ""|"localhost"|"127.0.0.1"|"::1")
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

append_no_proxy() {
  local item="$1"
  local current="${NO_PROXY:-${no_proxy:-}}"
  [[ -n "${item// }" ]] || return 0
  case ",$current," in
    *",$item,"*) ;;
    *)
      if [[ -z "$current" ]]; then
        current="$item"
      else
        current="$current,$item"
      fi
      ;;
  esac
  export NO_PROXY="$current"
  export no_proxy="$current"
}

append_no_proxy_for_url() {
  local url="$1"
  local host
  host="$(url_host "$url")"
  if is_loopback_host "$host"; then
    append_no_proxy "$host"
  fi
}

should_bypass_proxy_for_url() {
  local url="$1"
  local host
  host="$(url_host "$url")"
  is_loopback_host "$host"
}

enable_china_hf_endpoint() {
  local enabled="${1:-0}"
  if (( enabled )) && [[ -z "${HF_ENDPOINT:-}" ]]; then
    export HF_ENDPOINT="https://hf-mirror.com"
    echo "using HF_ENDPOINT=https://hf-mirror.com for this run (--china); profile files and .env.local are not modified"
  fi
}
