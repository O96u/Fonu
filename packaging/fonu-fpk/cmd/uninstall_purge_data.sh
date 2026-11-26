# shellcheck shell=bash
# Sourced by uninstall_init / uninstall_callback when the user chooses to delete all data.

# Only package-declared leaf shares may be removed. Never delete their parent.
is_fonu_share_path() {
  local p="$1"
  [[ "$p" = /* && "$p" != *'/../'* && "$p" != *'/./'* && "$p" != *'//'* ]] || return 1
  case "$p" in
    */fonu/etc|*/fonu/www|*/fonu/certs) ;;
    *) return 1 ;;
  esac
  # A symlinked parent must still resolve to the declared package share.
  local parent
  parent="$(dirname -- "$p")"
  if [ -e "$parent" ]; then
    local resolved
    resolved="$(readlink -f -- "$parent")" || return 1
    [ "$(basename -- "$resolved")" = fonu ] || return 1
  fi
  return 0
}

purge_fonu_user_data() {
  local log="${TRIM_TEMP_LOGFILE:-/tmp/fonu-uninstall.log}"
  local app_name="${TRIM_APPNAME:-fonu}"
  [[ "$app_name" =~ ^[a-zA-Z0-9_-]+$ ]] || return 1
  local app_root="/var/apps/${app_name}"

  log_msg() {
    printf '%s - %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$1" >>"$log"
  }

  log_msg "purge: start (wizard_delete_data=${wizard_delete_data:-})"

  for dir in "${TRIM_PKGVAR:-}" "${TRIM_PKGETC:-}" "${app_root}/var" "${app_root}/etc"; do
    if [ -z "$dir" ]; then continue; fi
    [[ "$dir" = /* && "$dir" != *'/../'* && "$dir" != *'/./'* && "$dir" != *'//'* && "$dir" != */.. && "$dir" != */. ]] || { log_msg "purge: refused invalid path"; continue; }
    case "$dir" in
      "$app_root/var"|"$app_root/etc") ;;
      */"$app_name"/var|*/"$app_name"/etc|*/"$app_name")
        [ "$dir" != "$app_root" ] || { log_msg "purge: refused app root"; continue; } ;;
      *) log_msg "purge: refused unrecognized package directory"; continue ;;
    esac
    if rm -rf -- "$dir" 2>>"$log"; then
      log_msg "purge: removed ${dir}"
    else
      log_msg "purge: failed ${dir} (see stderr in log)"
    fi
  done

  for share_root in "${app_root}/shares/fonu" "${app_root}/share/fonu"; do
    for leaf in etc www certs; do
      if is_fonu_share_path "$share_root/$leaf"; then
        rm -rf -- "$share_root/$leaf" 2>>"$log" || true
      fi
    done
    rmdir -- "$share_root" 2>/dev/null || true
  done

  if [ -n "${TRIM_DATA_SHARE_PATHS:-}" ]; then
    local -a paths=()
    IFS=':' read -r -a paths <<<"${TRIM_DATA_SHARE_PATHS}"
    for p in "${paths[@]}"; do
      [ -n "$p" ] || continue
      if ! is_fonu_share_path "$p"; then log_msg "purge: refused undeclared share"; continue; fi
      if rm -rf -- "$p" 2>>"$log"; then
        log_msg "purge: removed share ${p}"
      else
        log_msg "purge: failed share ${p}"
      fi
    done
  fi

  log_msg "purge: done"
}

should_purge_fonu_data() {
  case "${wizard_delete_data:-no}" in
    yes|true|1|on|Yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}
