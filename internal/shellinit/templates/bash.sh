idk() {
  local __idk_out
  __idk_out="$(command idk-bin "$@")" || return $?
  local __idk_op="${__idk_out%%$'\t'*}"
  local __idk_cmd="${__idk_out#*$'\t'}"
  case "$__idk_op" in
    RUN)
      eval "$__idk_cmd"
      ;;
    EDIT)
      history -s -- "$__idk_cmd"
      printf 'idk: press Up-arrow then Enter (or edit) to run:\n  %s\n' "$__idk_cmd" >&2
      ;;
    *) ;;
  esac
}

if [[ -z "$__IDK_HOOK_INSTALLED" ]]; then
  __IDK_HOOK_INSTALLED=1
  __idk_log_file="${XDG_DATA_HOME:-$HOME/.local/share}/idk/history.log"
  mkdir -p -- "$(dirname -- "$__idk_log_file")" 2>/dev/null
  if ((BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 2))); then
    __idk_ts() { printf -v __idk_ts_out '%(%s)T' -1; }
  else
    __idk_ts() { __idk_ts_out="$(date +%s)"; }
  fi

  __idk_precmd() {
    local __idk_exit=$?
    local raw cmd
    raw="$(HISTTIMEFORMAT= builtin history 1)"
    cmd="${raw#*[0-9] }"
    cmd="${cmd#"${cmd%%[![:space:]]*}"}"
    if [[ -n "$cmd" && "$cmd" != "$__idk_prev_cmd" && "$cmd" != idk\ * && "$cmd" != "idk" ]]; then
      __idk_prev_cmd="$cmd"
      __idk_ts
      printf '%s\t%s\t%s\n' "$__idk_ts_out" "$PWD" "$cmd" >> "$__idk_log_file"
    fi
    return $__idk_exit
  }
  case ";${PROMPT_COMMAND:-};" in
    *";__idk_precmd;"*) ;;
    *) PROMPT_COMMAND="__idk_precmd${PROMPT_COMMAND:+; $PROMPT_COMMAND}" ;;
  esac
fi
