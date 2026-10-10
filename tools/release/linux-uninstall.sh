#!/bin/sh
set -eu
install_root="$HOME/.local/lib/learn-with-ai"
launcher="$HOME/.local/bin/learn-with-ai"
desktop="${XDG_DATA_HOME:-$HOME/.local/share}/applications/learn-with-ai.desktop"
if [ -L "$install_root" ] || [ -L "$install_root/.learn-with-ai-files" ] || [ ! -f "$install_root/.learn-with-ai-files" ]; then
  echo 'Installation manifest unavailable or linked.' >&2; exit 1
fi
while IFS= read -r relative; do
  case "$relative" in app/*) ;; *) echo 'Invalid installer manifest.' >&2; exit 1 ;; esac
  case "$relative" in *..*|*\\*) echo 'Unsafe installer path.' >&2; exit 1 ;; esac
  target="$install_root/$relative"
  parent=$(dirname "$target")
  while [ "$parent" != / ]; do
    if [ -L "$parent" ]; then echo 'Refusing linked installation parent.' >&2; exit 1; fi
    parent=$(dirname "$parent")
  done
  if [ -L "$target" ]; then echo 'Refusing linked application file.' >&2; exit 1; fi
  rm -f -- "$target"
  directory=$(dirname "$target")
  while [ "$directory" != "$install_root" ]; do
    rmdir -- "$directory" 2>/dev/null || break
    directory=$(dirname "$directory")
  done
done < "$install_root/.learn-with-ai-files"
for entry in "$launcher" "$desktop"; do
  if [ -f "$entry" ] && [ ! -L "$entry" ] && grep -q '# learn-with-ai managed' "$entry"; then rm -f -- "$entry"; fi
done
rm -f -- "$install_root/.learn-with-ai-files" "$install_root/uninstall.sh"
rmdir -- "$install_root" 2>/dev/null || true
echo 'Uninstalled Learn-With-AI. Learning data is retained.'
