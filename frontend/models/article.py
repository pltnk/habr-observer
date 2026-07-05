from __future__ import annotations

import datetime
from dataclasses import dataclass

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

    @classmethod
    def from_dict(cls, d: dict) -> Article:
        return Article(
            _id=d["id"],
            title=d["title"],
            # Python 3.9's fromisoformat cannot parse the RFC 3339 "Z" suffix.
            pub_date=datetime.datetime.fromisoformat(d["pub_date"].replace("Z", "+00:00")),
            author=d["author"],
            summary=Summary.from_dict(d["summary"]),
        )
