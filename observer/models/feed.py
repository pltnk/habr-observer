from __future__ import annotations

from dataclasses import dataclass
from typing import List

from .article import Article


@dataclass
class Feed:
    _id: str
    name: str
    articles: List[Article]

    @classmethod
    def from_dict(cls, d: dict) -> Feed:
        return Feed(
            _id=d["id"],
            name=d["name"],
            articles=[Article.from_dict(a) for a in d["articles"]],
        )
