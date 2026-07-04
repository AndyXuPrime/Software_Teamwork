#!/usr/bin/env bash

prepare_knowledge_runtime_config() {
  local runtime_mode="${1:-host}"
  [[ "$runtime_mode" == "host" ]] || return 0
  [[ "$(to_lower "${DOC_ENGINE:-elasticsearch}")" == "elasticsearch" ]] || return 0
  [[ -n "${KNOWLEDGE_RUNTIME_ES_URL:-}" ]] || return 0
  if [[ "${RAGFLOW_CONF_EXPLICIT:-0}" == "1" && "${KNOWLEDGE_RUNTIME_GENERATE_LOCAL_CONF:-0}" != "1" ]]; then
    echo "using explicit RAGFLOW_CONF=$RAGFLOW_CONF; ensure its es.hosts matches $KNOWLEDGE_RUNTIME_ES_URL"
    return 0
  fi

  CURRENT_STEP="preparing knowledge-runtime config"
  local config_file="$LOCAL_RUNTIME_DIR/service_conf.yaml"
  mkdir -p "$LOCAL_RUNTIME_DIR"
  awk -v es_url="$KNOWLEDGE_RUNTIME_ES_URL" '
    BEGIN { in_es = 0; replaced = 0 }
    /^es:[[:space:]]*$/ { in_es = 1; print; next }
    /^[^[:space:]][^:]*:/ {
      if (in_es && !replaced) {
        print "  hosts: " es_url
        replaced = 1
      }
      in_es = 0
    }
    in_es && /^[[:space:]]+hosts:[[:space:]]*/ {
      print "  hosts: " es_url
      replaced = 1
      next
    }
    { print }
    END {
      if (in_es && !replaced) {
        print "  hosts: " es_url
      }
    }
  ' "$RUNTIME_DIR/conf/service_conf.yaml" >"$config_file"
  export RAGFLOW_CONF="$config_file"
  echo "knowledge-runtime config generated: $RAGFLOW_CONF"
}
