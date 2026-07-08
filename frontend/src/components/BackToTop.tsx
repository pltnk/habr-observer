import { useEffect, useState } from "react";
import { CircleArrowUp } from "@gravity-ui/icons";
import { Button, Icon, Tooltip } from "@gravity-ui/uikit";

// Show the button once the page is scrolled at least this far from the top.
const SCROLL_THRESHOLD_PX = 150;

/**
 * Back-to-top button, fixed bottom-right. Hidden at the top of the page; it
 * slides up into view once the page is scrolled down and slides back down when
 * it returns to the top (via the button or by scrolling). Scrolls directly
 * rather than to a URL hash, which would pollute analytics URL stats.
 */
export function BackToTop() {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    let ticking = false;
    const update = () => {
      setVisible(window.scrollY > SCROLL_THRESHOLD_PX);
      ticking = false;
    };
    const onScroll = () => {
      // Coalesce scroll events to one state read per frame.
      if (!ticking) {
        ticking = true;
        requestAnimationFrame(update);
      }
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    update(); // honor a scroll position restored on load
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <div
      className={
        visible
          ? "back-to-top-wrap back-to-top-wrap-visible"
          : "back-to-top-wrap"
      }
    >
      <Tooltip content="Вернуться в начало">
        <Button
          view="flat-secondary"
          size="xl"
          pin="circle-circle"
          aria-label="Вернуться в начало"
          onClick={() => window.scrollTo(0, 0)}
        >
          <Icon data={CircleArrowUp} size={32} />
        </Button>
      </Tooltip>
    </div>
  );
}
