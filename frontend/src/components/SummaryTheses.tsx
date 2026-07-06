import { useId, useState } from "react";
import { ChevronDownWide } from "@gravity-ui/icons";
import { Button, Icon, Tooltip } from "@gravity-ui/uikit";

// Mirrors the original's visible_theses=3: with collapsing on, this many
// theses stay visible and the rest hide behind the curtain.
const VISIBLE_THESES = 3;

interface SummaryThesesProps {
  content: string[];
  collapsed: boolean;
}

// `last` marks the list that ends the card, giving it the extra bottom
// margin that evens out the whitespace before the divider.
function ThesesList({
  theses,
  last = false,
}: {
  theses: string[];
  last?: boolean;
}) {
  return (
    <ul className={last ? "theses theses-last" : "theses"}>
      {theses.map((thesis, index) => (
        <li key={index}>{thesis}</li>
      ))}
    </ul>
  );
}

// Hand-rolled curtain disclosure: Gravity has no height-animating collapse
// (Disclosure fades opacity, Accordion disables even that), and neither can
// place the trigger below the content. Here the toggle strip follows the
// curtain region, so expanding slides it down past the revealed theses and
// it always closes the card instead of splitting the list.
export function SummaryTheses({ content, collapsed }: SummaryThesesProps) {
  const curtainId = useId();
  const toggleId = useId();
  const [expanded, setExpanded] = useState(false);

  if (!collapsed || content.length <= VISIBLE_THESES) {
    return <ThesesList theses={content} last />;
  }
  const label = expanded
    ? "Свернуть продолжение пересказа"
    : "Развернуть продолжение пересказа";
  return (
    <>
      <ThesesList theses={content.slice(0, VISIBLE_THESES)} />
      <div
        id={curtainId}
        role="region"
        aria-labelledby={toggleId}
        className={
          expanded ? "theses-curtain theses-curtain-open" : "theses-curtain"
        }
      >
        <div>
          <ThesesList theses={content.slice(VISIBLE_THESES)} />
        </div>
      </div>
      <Tooltip content={label}>
        <Button
          view="flat-secondary"
          size="l"
          width="max"
          id={toggleId}
          aria-controls={curtainId}
          aria-expanded={expanded}
          aria-label={label}
          onClick={() => setExpanded((value) => !value)}
          className="theses-toggle"
        >
          {/* One chevron for both states — CSS turns it 180° on expand,
              like Gravity's own Disclosure arrow. */}
          <Icon data={ChevronDownWide} size={20} />
        </Button>
      </Tooltip>
    </>
  );
}
