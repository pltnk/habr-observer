import asyncio
from typing import List, Optional

import httpx
from bs4 import BeautifulSoup

from models import Summary


async def get_summary_url(
    article_url: str,
    auth_token: str,
    lock: Optional[asyncio.Lock] = None,
    timeout: int = 4,
    is_retry: bool = False,
) -> Optional[str]:
    if lock:
        await lock.acquire()
        await asyncio.sleep(timeout)
    async with httpx.AsyncClient(
        headers={"Authorization": f"OAuth {auth_token}"}, timeout=20
    ) as client:
        res = await client.post(
            "https://300.ya.ru/api/sharing-url", json={"article_url": article_url}
        )
    if lock:
        lock.release()
    if res.status_code == 404:
        return None
    if res.status_code == 429 and not is_retry:
        return await get_summary_url(
            article_url=article_url,
            auth_token=auth_token,
            lock=lock,
            timeout=timeout * 2,
            is_retry=True,
        )
    res.raise_for_status()
    parsed = res.json()
    return parsed["sharing_url"]


async def get_summary_content_api(summary_url: str) -> List[str]:
    token = summary_url.rsplit("/", 1)[-1]
    async with httpx.AsyncClient(timeout=10) as client:
        res = await client.post("https://300.ya.ru/api/sharing", json={"token": token})
    res.raise_for_status()
    parsed = res.json()
    return [i["content"] for i in parsed["thesis"]]


async def get_summary_content_noapi(summary_url: str) -> List[str]:
    async with httpx.AsyncClient(timeout=10) as client:
        res = await client.get(summary_url)
    res.raise_for_status()
    parsed = BeautifulSoup(res.content, features="lxml")
    tag = parsed.find(
        "ul",
        attrs={"class": lambda c: isinstance(c, str) and c.startswith("theses-list")},
    )
    return [i.text.strip("• \n") for i in tag.find_all("li")]


async def get_summary_content(summary_url: str) -> List[str]:
    try:
        content = await get_summary_content_api(summary_url=summary_url)
    except Exception:
        content = await get_summary_content_noapi(summary_url=summary_url)
    return content


async def get_summary(
    article_url: str, auth_token: str, lock: Optional[asyncio.Lock] = None
) -> Summary:
    summary_url = await get_summary_url(
        article_url=article_url, auth_token=auth_token, lock=lock
    )
    if summary_url is None:
        return Summary(
            url="https://300.ya.ru",
            content=[
                "Статья слишком длинная, нейросети пока не умеют пересказывать такие статьи 😔"
            ],
        )
    content = await get_summary_content(summary_url=summary_url)
    return Summary(url=summary_url, content=content)
