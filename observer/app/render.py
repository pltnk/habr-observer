from typing import Iterable

import streamlit as st
from streamlit.components.v1 import html

from models import Article, Feed


def render_header() -> None:
    st.markdown(
        """
        <div id='top' style='text-align: center'>
        <p id='#top'></p>
        <br>
        <h1>🧐<br>Обозреватель Хабра</h1>
        <h2>Краткий пересказ лучших статей с Хабра от нейросети YandexGPT</h2>
        </div>
        """,
        unsafe_allow_html=True,
    )


def render_toggle() -> bool:
    st.write(
        """
        <style>
        label[data-baseweb="checkbox"] {
        justify-content: center;
        align-items: center;
        }
        </style>
        """,
        unsafe_allow_html=True,
    )
    return st.toggle(
        label="Сворачивать пересказы",
        value=True,
        key="collapse_summaries",
        help="Отключите, чтобы показывать пересказы целиком, не сворачивая их",
    )


def render_theses(theses: Iterable[str]) -> None:
    st.markdown("\n".join("* " + i for i in theses))


def render_article(
    article: Article, collapse_summary: bool = True, visible_theses: int = 3
) -> None:
    with st.container():
        st.subheader(
            article.title,
            help=f"Дата публикации: {article.pub_date}",
            anchor=False,
        )
        if collapse_summary:
            render_theses(article.summary.content[:visible_theses])
            if len(article.summary.content) > visible_theses:
                with st.expander(label="Продолжение пересказа"):
                    render_theses(article.summary.content[visible_theses:])
        else:
            render_theses(article.summary.content)
        st.caption(
            f"""
            <div style='text-align: center'>
            <a href='{article.summary.url}' target='_blank' style='text-decoration: none; color: inherit;'>
            🤖 Ссылка на пересказ
            </a>
            &emsp;&emsp;
            <a href='{article.url}' target='_blank' style='text-decoration: none; color: inherit;'>
            📃 Открыть оригинал
            </a>
            </div>
            """,
            unsafe_allow_html=True,
        )
        st.divider()


def render_tab(
    tab: st.delta_generator.DeltaGenerator,
    articles: Iterable[Article],
    collapse_summaries: bool = True,
) -> None:
    with tab:
        for a in articles:
            render_article(article=a, collapse_summary=collapse_summaries)


def render_tabs(feeds: Iterable[Feed], collapse_summaries: bool = True) -> None:
    st.write(
        """
        <style>
        div[data-baseweb="tab-list"] {
        justify-content: center;
        align-items: center;
        }
        </style>
        """,
        unsafe_allow_html=True,
    )
    tabs = st.tabs([feed.name for feed in feeds])
    for tab, feed in zip(tabs, feeds):
        render_tab(
            tab=tab, articles=feed.articles, collapse_summaries=collapse_summaries
        )

    # see for an explanation of the below code:
    # https://discuss.streamlit.io/t/bug-with-st-tabs-glitches-for-1-frame-while-rendering/33497/12
    html(
        """
        <script>
        function checkElements() {
        
            const tabs = window.parent.document.querySelectorAll('button[data-baseweb="tab"] p');
            const tab_panels = window.parent.document.querySelectorAll('div[data-baseweb="tab-panel"]');
        
            if (tabs && tab_panels) {
        
                tabs.forEach(function(tab, index) {
                    const tab_panel_child = tab_panels[index].querySelectorAll("*");
        
                    function set_visibility(state) {
                        tab_panels[index].style.visibility = state;
                        tab_panel_child.forEach(function(child) {
                            child.style.visibility = state;
                        });
                    }
        
                    tab.addEventListener("click", function(event) {
                        set_visibility('hidden')
        
                        let element = tab_panels[index].querySelector('div[data-testid="stVerticalBlock"]');
                        let main_block = window.parent.document.querySelector('section.main div[data-testid="stVerticalBlock"]');
                        const waitMs = 1;
        
                        function waitForLayout() {
                            if (element.offsetWidth === main_block.offsetWidth) {
                                set_visibility("visible");
                            } else {
                                setTimeout(waitForLayout, waitMs);
                            }
                        }
        
                        waitForLayout();
                    });
                });
            } else {
                setTimeout(checkElements, 50);
            }
        }
        
        checkElements();
        </script>
        """,
        height=0,
    )


def render_footer() -> None:
    st.caption(
        """
        <div style='text-align: center'>
        <a href='https://pltnk.dev' target='_blank' style='text-decoration: none; color: inherit;'>
        😎 Автор pltnk.dev
        </a>
        &emsp;&emsp;
        <a href='https://github.com/pltnk/habr-observer' target='_blank' style='text-decoration: none; color: inherit;'>
        🍝 Код на GitHub
        </a>
        </div>
        """,
        unsafe_allow_html=True,
    )
    st.caption(
        """
        <div style='text-align: center'>
        В приложении используются материалы сайта 
        <a href='https://habr.com' target='_blank' style='text-decoration: none; color: inherit;'>
        habr.com</a>, краткие пересказы которых получены с помощью сервиса
        <a href='https://300.ya.ru' target='_blank' style='text-decoration: none; color: inherit;'>
        300.ya.ru</a>.
        </div>
        """,
        unsafe_allow_html=True,
    )
    st.markdown(
        """
        <div style='position: fixed; bottom: 0px; right: 5px; font-size: xx-large;'>
        <a href='#top' style='text-decoration: none;'>⬆️</a>
        </div>
        """,
        unsafe_allow_html=True,
    )
