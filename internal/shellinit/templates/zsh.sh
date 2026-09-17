idk() {
  local __idk_out
  __idk_out="$(command idk-bin "$@")" || return $?
  local __idk_op="${__idk_out%%$'\t'*}"
  local __idk_cmd="${__idk_out#*$'\t'}"
  case "$__idk_op" in
    RUN)   eval "$__idk_cmd" ;;
    EDIT)  print -z -- "$__idk_cmd" ;;
    *)     ;;
  esac
}

if [[ -z "$__IDK_HOOK_INSTALLED" ]]; then
  __IDK_HOOK_INSTALLED=1
  zmodload zsh/datetime 2>/dev/null
  typeset -g __idk_log_file="${XDG_DATA_HOME:-$HOME/.local/share}/idk/history.log"
  mkdir -p -- "${__idk_log_file:h}" 2>/dev/null

  __idk_preexec() {
    case "$1" in
      idk|idk\ *|idk-bin|idk-bin\ *) __idk_pending="" ;;
      *) __idk_pending="$1" ;;
    esac
  }
  __idk_precmd() {
    if [[ -n "$__idk_pending" ]]; then
      print -r -- "${EPOCHSECONDS:-$(date +%s)}"$'\t'"$PWD"$'\t'"$__idk_pending" >> "$__idk_log_file"
      __idk_pending=""
    fi
  }
  autoload -Uz add-zsh-hook
  add-zsh-hook preexec __idk_preexec
  add-zsh-hook precmd  __idk_precmd
fi
