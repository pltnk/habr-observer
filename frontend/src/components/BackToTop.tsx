import { CircleArrowUp } from "@gravity-ui/icons";
import { Button, Icon, Tooltip } from "@gravity-ui/uikit";

// A plain in-page scroll action: no URL hash (it polluted Metrika's URL
// stats) and an absolute top, unlike the old #top anchor jump.
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
