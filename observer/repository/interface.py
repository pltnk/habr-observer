from abc import ABC, abstractmethod
from typing import Iterable

from models import Article, Feed


class AsyncRepository(ABC):
    @abstractmethod
    async def get_articles(self, urls: Iterable[str]) -> Iterable[Article]:
        raise NotImplementedError

    @abstractmethod
    async def insert_articles(self, articles: Iterable[Article]) -> None:
        raise NotImplementedError

    @abstractmethod
    async def insert_feed(self, feed: Feed) -> None:
        raise NotImplementedError

    @abstractmethod
    async def get_feeds(self, ids: Iterable[str]) -> Iterable[Feed]:
        raise NotImplementedError
