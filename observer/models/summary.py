from __future__ import annotations

from dataclasses import dataclass
from typing import List


@dataclass
class Summary:
    url: str
    content: List[str]

    @classmethod
    def from_dict(cls, d: dict) -> Summary:
        return Summary(url=d["url"], content=d["content"])
