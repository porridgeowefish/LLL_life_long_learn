#!/bin/sh
set -eu
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_root="$HOME/.local/lib/learn-with-ai"
launcher="$HOME/.local/bin/learn-with-ai"
desktop="${XDG_DATA_HOME:-$HOME/.local/share}/applications/learn-with-ai.desktop"
if [ -L "$install_root" ] || [ -L "$launcher" ] || [ -L "$desktop" ] || [ -L "$install_root/.learn-with-ai-files" ]; then
  echo 'Refusing linked installation paths.' >&2; exit 1
fi
if [ -e "$install_root" ] && [ ! -f "$install_root/.learn-with-ai-files" ]; then
  echo 'Install folder is not owned by learn-with-ai.' >&2; exit 1
fi
for entry in "$launcher" "$desktop"; do
  if [ -e "$entry" ] && ! grep -q '# learn-with-ai managed' "$entry"; then
    echo "Refusing to overwrite an unowned launcher: $entry" >&2; exit 1
  fi
done
mkdir -p "$install_root" "$(dirname "$launcher")" "$(dirname "$desktop")"
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
  if [ -e "$target" ] && ! grep -Fxq "$relative" "$install_root/.learn-with-ai-files"; then
    echo "Refusing to overwrite an unowned application file: $relative" >&2; exit 1
  fi
done < "$source_dir/.learn-with-ai-files"
cp -R "$source_dir/app" "$install_root/"
chmod +x "$install_root/app/learn-with-ai"
if [ -f "$install_root/.learn-with-ai-files" ]; then
  while IFS= read -r relative; do
    case "$relative" in app/*) ;; *) echo 'Invalid previous installer manifest.' >&2; exit 1 ;; esac
    case "$relative" in *..*|*\\*) echo 'Unsafe previous installer path.' >&2; exit 1 ;; esac
    if grep -Fxq "$relative" "$source_dir/.learn-with-ai-files"; then continue; fi
    target="$install_root/$relative"
    parent=$(dirname "$target")
    while [ "$parent" != / ]; do
      if [ -L "$parent" ]; then echo 'Refusing linked stale resource parent.' >&2; exit 1; fi
      parent=$(dirname "$parent")
    done
    if [ -L "$target" ]; then echo 'Refusing linked stale resource.' >&2; exit 1; fi
    rm -f -- "$target"
    directory=$(dirname "$target")
    while [ "$directory" != "$install_root" ]; do
      rmdir -- "$directory" 2>/dev/null || break
      directory=$(dirname "$directory")
    done
  done < "$install_root/.learn-with-ai-files"
fi
cp "$source_dir/uninstall.sh" "$install_root/uninstall.sh"
cp "$source_dir/.learn-with-ai-files" "$install_root/.learn-with-ai-files"
chmod +x "$install_root/uninstall.sh"
# Quote the actual home path for the shell launcher and desktop Exec grammar.
escaped_root=$(printf '%s' "$install_root" | sed "s/'/'\\\\''/g")
printf "#!/bin/sh\n# learn-with-ai managed launcher\nexec '%s/app/learn-with-ai' \"\$@\"\n" "$escaped_root" > "$launcher"
chmod +x "$launcher"
desktop_exec=$(printf '%s' "$install_root/app/learn-with-ai" | sed 's/\\/\\\\/g;s/"/\\"/g;s/`/\\`/g;s/\$/\\$/g')
printf '[Desktop Entry]\n# learn-with-ai managed desktop\nType=Application\nName=Learn-With-AI\nExec="%s"\nTerminal=false\nCategories=Education;\n' "$desktop_exec" > "$desktop"
echo "Installed Learn-With-AI. Open it from the applications menu or $launcher"
echo "Uninstall: $install_root/uninstall.sh (learning data is retained)."
