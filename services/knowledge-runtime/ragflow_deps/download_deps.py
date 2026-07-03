#!/usr/bin/env python3

# PEP 723 metadata
# /// script
# requires-python = ">=3.10"
# dependencies = [
#   "nltk",
#   "huggingface-hub"
# ]
# ///

# This script downloads runtime artifacts into `ragflow_deps/` for host-run
# Knowledge runtime development. Run it from anywhere: the `__main__` block
# chdir's into this file's own directory, so all outputs land under
# `ragflow_deps/` regardless of the caller's CWD.
#
# Typical workflow:
#
#   uv run ragflow_deps/download_deps.py
#   uv run ragflow_deps/download_deps.py --china-mirrors

import argparse
import os
import urllib.request
from typing import Union

from nltk.downloader import Downloader
from huggingface_hub import snapshot_download

GITHUB_PROXY_PREFIX = "https://gh-proxy.com/"
HF_MIRROR_ENDPOINT = "https://hf-mirror.com"
NLTK_DATA_INDEX_URL = "https://raw.githubusercontent.com/nltk/nltk_data/gh-pages/index.xml"
NLTK_DATA_MIRROR_INDEX_URL = f"{GITHUB_PROXY_PREFIX}{NLTK_DATA_INDEX_URL}"
NLTK_DATA_PACKAGE_PREFIX = "https://raw.githubusercontent.com/nltk/nltk_data/gh-pages/"
NLTK_DATA_MIRROR_PACKAGE_PREFIX = f"{GITHUB_PROXY_PREFIX}{NLTK_DATA_PACKAGE_PREFIX}"


def get_urls(use_china_mirrors=False) -> list[Union[str, list[str]]]:
    if use_china_mirrors:
        return [
            "http://mirrors.tuna.tsinghua.edu.cn/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb",
            "http://mirrors.tuna.tsinghua.edu.cn/ubuntu-ports/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_arm64.deb",
            "https://repo.huaweicloud.com/repository/maven/org/apache/tika/tika-server-standard/3.3.0/tika-server-standard-3.3.0.jar",
            "https://repo.huaweicloud.com/repository/maven/org/apache/tika/tika-server-standard/3.3.0/tika-server-standard-3.3.0.jar.md5",
            "https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken",
            ["https://registry.npmmirror.com/-/binary/chrome-for-testing/121.0.6167.85/linux64/chrome-linux64.zip", "chrome-linux64-121-0-6167-85"],
            ["https://registry.npmmirror.com/-/binary/chrome-for-testing/121.0.6167.85/linux64/chromedriver-linux64.zip", "chromedriver-linux64-121-0-6167-85"],
            f"{GITHUB_PROXY_PREFIX}https://github.com/astral-sh/uv/releases/download/0.9.16/uv-x86_64-unknown-linux-gnu.tar.gz",
            f"{GITHUB_PROXY_PREFIX}https://github.com/astral-sh/uv/releases/download/0.9.16/uv-aarch64-unknown-linux-gnu.tar.gz",
        ]
    else:
        return [
            "http://archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb",
            "http://ports.ubuntu.com/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_arm64.deb",
            "https://repo1.maven.org/maven2/org/apache/tika/tika-server-standard/3.3.0/tika-server-standard-3.3.0.jar",
            "https://repo1.maven.org/maven2/org/apache/tika/tika-server-standard/3.3.0/tika-server-standard-3.3.0.jar.md5",
            "https://openaipublic.blob.core.windows.net/encodings/cl100k_base.tiktoken",
            ["https://storage.googleapis.com/chrome-for-testing-public/121.0.6167.85/linux64/chrome-linux64.zip", "chrome-linux64-121-0-6167-85"],
            ["https://storage.googleapis.com/chrome-for-testing-public/121.0.6167.85/linux64/chromedriver-linux64.zip", "chromedriver-linux64-121-0-6167-85"],
            "https://github.com/astral-sh/uv/releases/download/0.9.16/uv-x86_64-unknown-linux-gnu.tar.gz",
            "https://github.com/astral-sh/uv/releases/download/0.9.16/uv-aarch64-unknown-linux-gnu.tar.gz",
        ]


repos = [
    "InfiniFlow/text_concat_xgb_v1.0",
    "InfiniFlow/deepdoc",
]


def download_model(repository_id):
    local_directory = os.path.abspath(os.path.join("huggingface.co", repository_id))
    os.makedirs(local_directory, exist_ok=True)
    endpoint = HF_MIRROR_ENDPOINT if os.environ.get("RAGFLOW_USE_CHINA_MIRRORS") == "1" else None
    snapshot_download(repo_id=repository_id, local_dir=local_directory, endpoint=endpoint)


def build_nltk_downloader(use_china_mirrors=False):
    index_url = os.environ.get("NLTK_DOWNLOAD_INDEX_URL")
    if not index_url:
        index_url = NLTK_DATA_MIRROR_INDEX_URL if use_china_mirrors else NLTK_DATA_INDEX_URL

    downloader = Downloader(server_index_url=index_url)
    if use_china_mirrors or index_url.startswith(GITHUB_PROXY_PREFIX):
        package_prefix = os.environ.get("NLTK_DOWNLOAD_PACKAGE_PREFIX", NLTK_DATA_MIRROR_PACKAGE_PREFIX)
        # NLTK stores package URLs inside index.xml. Rewrite those URLs after
        # loading the index so the actual zip downloads also use the mirror.
        downloader._update_index()
        for package in downloader._packages.values():
            if package.url.startswith(NLTK_DATA_PACKAGE_PREFIX):
                package.url = package_prefix + package.url[len(NLTK_DATA_PACKAGE_PREFIX):]
    return downloader


if __name__ == "__main__":
    # Anchor CWD to this file's directory so all relative outputs
    # (huggingface.co/, nltk_data/, *.deb, *.jar, *.tar.gz, etc.) land
    # at the top of ragflow_deps/ regardless of where the user invokes
    # the script from.
    os.chdir(os.path.dirname(os.path.abspath(__file__)))

    parser = argparse.ArgumentParser(description="Download dependencies with optional China mirror support")
    parser.add_argument("--china-mirrors", action="store_true", help="Use China-accessible mirrors for downloads")
    args = parser.parse_args()

    urls = get_urls(args.china_mirrors)
    if args.china_mirrors:
        os.environ.setdefault("RAGFLOW_USE_CHINA_MIRRORS", "1")

    # Some mirrors (e.g. archive.ubuntu.com) reject the default urllib
    # User-Agent with HTTP 403, so install an opener with a browser-like UA.
    opener = urllib.request.build_opener()
    opener.addheaders = [("User-Agent", "Mozilla/5.0")]
    urllib.request.install_opener(opener)

    for url in urls:
        download_url = url[0] if isinstance(url, list) else url
        filename = url[1] if isinstance(url, list) else url.split("/")[-1]
        print(f"Downloading {filename} from {download_url}...")
        if not os.path.exists(filename):
            urllib.request.urlretrieve(download_url, filename)

    local_dir = os.path.abspath("nltk_data")
    nltk_downloader = build_nltk_downloader(args.china_mirrors)
    for data in ["wordnet", "punkt", "punkt_tab"]:
        print(f"Downloading nltk {data}...")
        nltk_downloader.download(data, download_dir=local_dir)

    for repo_id in repos:
        print(f"Downloading huggingface repo {repo_id}...")
        download_model(repo_id)
