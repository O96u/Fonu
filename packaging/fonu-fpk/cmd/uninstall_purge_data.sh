# shellcheck shell=bash
# Sourced by uninstall_init / uninstall_callback when the user chooses to delete all data.

purge_fonu_user_data() {
  local log="${TRIM_TEMP_LOGFILE:-/tmp/fonu-uninstall.log}"
  local app_root="/var/apps/${TRIM_APPNAME:-fonu}"

  log_msg() {
    printf '%s - %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$1" >>"$log"
  }

  log_msg "purge: start (wizard_delete_data=${wizard_delete_data:-})"

  for dir in "${TRIM_PKGVAR:-}" "${TRIM_PKGETC:-}" "${app_root}/var" "${app_root}/etc"; do
    if [ -z "$dir" ]; then
      continue
    fi
    if rm -rf "$dir" 2>>"$log"; then
      log_msg "purge: removed ${dir}"
    else
      log_msg "purge: failed ${dir} (see stderr in log)"
    fi
  done

  rm -rf "${app_root}/shares/fonu" "${app_root}/share/fonu" 2>>"$log" || true

  if [ -n "${TRIM_DATA_SHARE_PATHS:-}" ]; then
    local -a paths=()
    IFS=':' read -r -a paths <<<"${TRIM_DATA_SHARE_PATHS}"
    local parent=""
    for p in "${paths[@]}"; do
      [ -n "$p" ] || continue
      parent="$(dirname "$p")"
      if rm -rf "$p" 2>>"$log"; then
        log_msg "purge: removed share ${p}"
      else
        log_msg "purge: failed share ${p}"
      fi
    done
    if [ -n "$parent" ] && [ "$(basename "$parent")" = "fonu" ]; then
      rm -rf "$parent" 2>>"$log" && log_msg "purge: removed ${parent}" || log_msg "purge: failed ${parent}"
    fi
  fi

  log_msg "purge: done"
}

should_purge_fonu_data() {
  case "${wizard_delete_data:-no}" in
    yes|true|1|on|Yes|YES) return 0 ;;
    *) return 1 ;;
  esac
}
