from __future__ import annotations

from dataclasses import dataclass, asdict
from typing import List


@dataclass
class Summary:
    url: str
    content: List[str]

    def as_dict(self) -> dict:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict) -> Summary:
        return Summary(url=d["url"], content=d["content"])
