from __future__ import annotations

import datetime
from dataclasses import dataclass, asdict

from .summary import Summary


@dataclass
class Article:
    _id: str
    title: str
    pub_date: datetime.datetime
    author: str
    summary: Summary

    @property
    def url(self) -> str:
        return self._id

    def as_dict(self) -> dict:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict) -> Article:
        return Article(
            _id=d["_id"],
            title=d["title"],
            pub_date=d["pub_date"],
            author=d["author"],
            summary=Summary.from_dict(d["summary"]),
        )
