import asyncio
import datetime
from typing import Dict, Optional, Iterable

import httpx
from bs4 import BeautifulSoup
from bs4.element import Tag

from models import Article, Feed
from repository import AsyncRepository
from .summary import get_summary

DT_FORMAT = "%a, %d %b %Y %H:%M:%S %Z"


class FeedUpdater:
    def __init__(
        self,
        name: str,
        url: str,
        summary_auth_token: str,
        repository: AsyncRepository,
        throttle_lock: Optional[asyncio.Lock] = None,
    ):
        self._name = name
        self._url = url
        self._summary_auth_token = summary_auth_token
        self._repository = repository
        self._throttle_lock = throttle_lock
        self._parsed: Optional[BeautifulSoup] = None
        self._article_urls: Optional[Iterable[str]] = None
        self._articles_present: Dict[str, Article] = {}
        self._articles_to_scrape: Optional[Iterable[Tag]] = None

    async def _get_feed_content(self) -> None:
        async with httpx.AsyncClient(timeout=10) as client:
            res = await client.get(self._url)
        res.raise_for_status()
        self._parsed = BeautifulSoup(res.content, features="xml")

    async def _filter_articles(self) -> None:
        self._article_urls = [i.text for i in self._parsed.find_all("guid")]
        articles = await self._repository.get_articles(self._article_urls)
        for a in articles:
            self._articles_present[a.url] = a
        self._articles_to_scrape = [
            tag
            for tag in self._parsed.find_all("item")
            if tag.find("guid").text not in self._articles_present
        ]

    async def _scrape_articles(self) -> None:
        tasks = (self._get_article(tag=tag) for tag in self._articles_to_scrape)
        result = await asyncio.gather(*tasks, return_exceptions=True)
        articles = [a for a in result if isinstance(a, Article)]
        await self._repository.insert_articles(articles)
        for a in articles:
            self._articles_present[a.url] = a

    async def _get_article(self, tag: Tag) -> Article:
        url = tag.find("guid").text
        summary = await get_summary(
            article_url=url,
            auth_token=self._summary_auth_token,
            lock=self._throttle_lock,
        )
        title = tag.find("title").text or "Без названия"
        pub_date = datetime.datetime.strptime(tag.find("pubdate").text, DT_FORMAT)
        author = tag.find("dc:creator").text
        return Article(
            _id=url, title=title, pub_date=pub_date, author=author, summary=summary
        )

    async def _insert_feed(self):
        feed = Feed(
            _id=self._url,
            name=self._name,
            articles=[self._articles_present[url] for url in self._article_urls],
        )
        await self._repository.insert_feed(feed)

    async def update_feed(self):
        await self._get_feed_content()
        await self._filter_articles()
        if self._articles_to_scrape:
            await self._scrape_articles()
        await self._insert_feed()
