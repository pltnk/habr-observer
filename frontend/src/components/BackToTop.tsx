import { Link } from "@gravity-ui/uikit";

// An instant in-page anchor jump, same as the original's fixed ⬆️ link.
export function BackToTop() {
  return (
    <Link href="#top" aria-label="Наверх" className="back-to-top">
      ⬆️
    </Link>
  );
}
