#!/usr/bin/env python3
"""Extract data.tar.* from a .deb (ar archive) into an output directory."""
import lzma
import os
import sys
import tarfile


def read_ar_member(f):
    hdr = f.read(60)
    if len(hdr) < 60:
        return None, None
    name = hdr[:16].decode("ascii", errors="replace").strip()
    size = int(hdr[48:58].decode("ascii"))
    payload = f.read(size)
    if size % 2:
        f.read(1)
    return name, payload


def extract_deb(deb_path: str, out_dir: str) -> None:
    os.makedirs(out_dir, exist_ok=True)
    with open(deb_path, "rb") as f:
        while True:
            name, payload = read_ar_member(f)
            if not name:
                break
            if not name.startswith("data.tar"):
                continue
            data_path = os.path.join(out_dir, name)
            with open(data_path, "wb") as w:
                w.write(payload)
            if name.endswith(".xz"):
                with lzma.open(data_path) as xz:
                    with tarfile.open(fileobj=xz) as tar:
                        tar.extractall(out_dir)
            else:
                with tarfile.open(data_path) as tar:
                    tar.extractall(out_dir)
            return
    raise SystemExit(f"no data.tar member in {deb_path}")


if __name__ == "__main__":
    extract_deb(sys.argv[1], sys.argv[2])
