import { CircleArrowUp } from "@gravity-ui/icons";
import { Icon, Link, Tooltip } from "@gravity-ui/uikit";

// The original's instant in-page anchor jump, with the parity-era ⬆️ emoji
// modernized to a Gravity icon post-parity.
export function BackToTop() {
  return (
    <Tooltip content="Вернуться в начало">
      <Link
        view="secondary"
        href="#top"
        aria-label="Вернуться в начало"
        className="back-to-top"
      >
        <Icon data={CircleArrowUp} size={32} />
      </Link>
    </Tooltip>
  );
}
