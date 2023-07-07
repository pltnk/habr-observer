from __future__ import annotations

from dataclasses import dataclass, asdict
from typing import List

from .article import Article


@dataclass
class Feed:
    _id: str
    name: str
    articles: List[Article]

    @property
    def url(self):
        return self._id

    def as_dict(self) -> dict:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict) -> Feed:
        return Feed(
            _id=d["_id"],
            name=d["name"],
            articles=[Article.from_dict(a) for a in d["articles"]],
        )
