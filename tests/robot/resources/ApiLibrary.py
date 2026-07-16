from __future__ import annotations

import json
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen


def get_text(url: str) -> str:
    with urlopen(url, timeout=5) as response:
        return response.read().decode("utf-8")


def get_json(url: str):
    with urlopen(url, timeout=5) as response:
        return json.loads(response.read().decode("utf-8"))


def load_json_file(path: str):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def post_json(url: str, payload):
    request = Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"content-type": "application/json"},
        method="POST",
    )
    with urlopen(request, timeout=5) as response:
        return json.loads(response.read().decode("utf-8"))


def post_json_accepted(url: str, payload):
    request = Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"content-type": "application/json"},
        method="POST",
    )
    with urlopen(request, timeout=5) as response:
        if response.status not in (200, 202):
            raise AssertionError(f"expected 202, got {response.status}")
        return json.loads(response.read().decode("utf-8"))


def post_json_expect_error(url: str, payload, expected_status: int):
    request = Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"content-type": "application/json"},
        method="POST",
    )
    try:
        with urlopen(request, timeout=5) as response:
            raise AssertionError(f"expected HTTP {expected_status}, got {response.status}")
    except HTTPError as exc:
        if exc.code != int(expected_status):
            raise AssertionError(f"expected HTTP {expected_status}, got {exc.code}") from exc
        return json.loads(exc.read().decode("utf-8"))
