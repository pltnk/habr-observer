import { Link, Text } from "@gravity-ui/uikit";

export function Footer() {
  return (
    <footer className="footer">
      <Text as="p" variant="caption-2" color="secondary" className="links-row">
        <Link
          view="secondary"
          href="https://pltnk.dev"
          target="_blank"
          rel="noopener noreferrer"
        >
          😎 Автор pltnk.dev
        </Link>
        <Link
          view="secondary"
          href="https://github.com/pltnk/habr-observer"
          target="_blank"
          rel="noopener noreferrer"
        >
          🍝 Код на GitHub
        </Link>
      </Text>
      <Text
        as="p"
        variant="caption-2"
        color="secondary"
        className="attribution"
      >
        В приложении используются материалы сайта{" "}
        <Link
          view="secondary"
          href="https://habr.com"
          target="_blank"
          rel="noopener noreferrer"
        >
          habr.com
        </Link>
        , краткие пересказы которых получены с помощью сервиса{" "}
        <Link
          view="secondary"
          href="https://300.ya.ru"
          target="_blank"
          rel="noopener noreferrer"
        >
          300.ya.ru
        </Link>
        .
      </Text>
    </footer>
  );
}
