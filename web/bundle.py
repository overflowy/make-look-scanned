#!/usr/bin/env python3
"""Inline the web shell into one self-contained, offline HTML file.

Reads web/index.html and replaces its external references — wasm_exec.js, the
PDF.js ESM module, its worker, and main.wasm — with inlined data, producing
dist/make-look-scanned.html. The big assets are base64'd into <script
type="text/plain"> blocks (base64 has no '<', so nothing can break out of the
script tag) and decoded to Blobs/bytes at runtime. The GLUE script in index.html
is reused verbatim; only the CONFIG block is swapped.
"""

import base64
import pathlib
import re
import sys
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parent.parent
WEB = ROOT / "web"
VENDOR = WEB / "vendor"
DIST = ROOT / "dist"
PDFJS_VER = "6.0.227"

# The library is the self-contained +esm bundle; the worker is the standalone
# build file. Vendor filenames carry the version so a bump re-downloads instead
# of reusing a stale cached copy.
PDFJS_LIB = f"pdfjs-{PDFJS_VER}.mjs"
PDFJS_WORKER = f"pdfworker-{PDFJS_VER}.mjs"
ASSETS = {
    PDFJS_LIB: f"https://cdn.jsdelivr.net/npm/pdfjs-dist@{PDFJS_VER}/+esm",
    PDFJS_WORKER: f"https://cdn.jsdelivr.net/npm/pdfjs-dist@{PDFJS_VER}/build/pdf.worker.min.mjs",
}


def b64(path: pathlib.Path) -> str:
    return base64.b64encode(path.read_bytes()).decode()


def main() -> None:
    if not (WEB / "main.wasm").exists() or not (WEB / "wasm_exec.js").exists():
        sys.exit("missing web/main.wasm or web/wasm_exec.js — run web/build.sh first")

    VENDOR.mkdir(exist_ok=True)
    for name, url in ASSETS.items():
        dst = VENDOR / name
        if not dst.exists():
            print("downloading", url)
            urllib.request.urlretrieve(url, dst)

    wasm_exec = (WEB / "wasm_exec.js").read_text()
    if "</script" in wasm_exec.lower():
        sys.exit("wasm_exec.js contains a literal </script> — cannot inline safely")

    html = (WEB / "index.html").read_text()

    # 1) Inline wasm_exec.js (executable; Go's runtime has no </script> token).
    tag = '<script src="wasm_exec.js"></script>'
    if tag not in html:
        sys.exit("could not find wasm_exec.js script tag in index.html")
    html = html.replace(tag, "<script>\n" + wasm_exec + "\n</script>")

    # 2) Replace the CONFIG block with inlined, base64'd assets.
    config = f"""<script type="text/plain" id="pdfjs-b64">{b64(VENDOR / PDFJS_LIB)}</script>
<script type="text/plain" id="worker-b64">{b64(VENDOR / PDFJS_WORKER)}</script>
<script type="text/plain" id="wasm-b64">{b64(WEB / "main.wasm")}</script>
<script id="mls-config">
  (() => {{
    const bytes = (id) => Uint8Array.from(atob(document.getElementById(id).textContent), (c) => c.charCodeAt(0));
    const blobUrl = (id) => URL.createObjectURL(new Blob([bytes(id)], {{ type: "text/javascript" }}));
    window.MLS = {{
      pdfjsModuleUrl: blobUrl("pdfjs-b64"),
      workerSrc: blobUrl("worker-b64"),
      wasmBytes: () => Promise.resolve(bytes("wasm-b64").buffer),
    }};
  }})();
</script>"""
    html, n = re.subn(
        r'<script id="mls-config">.*?</script>', lambda *_: config, html, count=1, flags=re.S
    )
    if n != 1:
        sys.exit(f"expected exactly one mls-config block, replaced {n}")

    DIST.mkdir(exist_ok=True)
    out = DIST / "make-look-scanned.html"
    out.write_text(html)
    print(
        f"wrote {out} ({out.stat().st_size:,} bytes) — open it directly in a browser, no server needed"
    )


if __name__ == "__main__":
    main()
