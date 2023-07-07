import streamlit as st

from app import run_app

st.set_page_config(
    page_title="habr.observer",
    page_icon="🧐",
    menu_items={
        "Get help": None,
        "Report a Bug": "https://github.com/pltnk/habr-observer/issues",
        "About": "Author: [pltnk.dev](https://pltnk.dev) ✧˚₊‧⋆ "
        "Code: [github.com/pltnk/habr-observer](https://github.com/pltnk/habr-observer)",
    },
)


if __name__ == "__main__":
    run_app()
