#!/bin/bash
# PreToolUse: bloquea acceso a archivos de secretos.
python3 -I -c '
import json,sys,re
d=json.load(sys.stdin)
p=d.get("tool_input",{}).get("file_path","") or ""
if re.search(r"(^|/)\.env(\.[^/]*)?$|\.pem$|\.key$|id_rsa|credentials", p) and not p.endswith(".env.example"):
    sys.stderr.write("Bloqueado: archivo de secretos (%s)\n" % p); sys.exit(2)
'
