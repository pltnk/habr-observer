import { CircleArrowUp } from "@gravity-ui/icons";
import { Button, Icon, Tooltip } from "@gravity-ui/uikit";

/**
 * Fixed button that scrolls the page to the top. Scrolls directly instead of
 * navigating to a URL hash, which would pollute analytics URL stats.
 */
export function BackToTop() {
  return (
    <Tooltip content="Вернуться в начало">
      <Button
        view="flat-secondary"
        size="xl"
        pin="circle-circle"
        aria-label="Вернуться в начало"
        className="back-to-top"
        onClick={() => window.scrollTo(0, 0)}
      >
        <Icon data={CircleArrowUp} size={32} />
      </Button>
    </Tooltip>
  );
}
