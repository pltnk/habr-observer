import { useId, useState } from "react";
import { ChevronDownWide } from "@gravity-ui/icons";
import { Button, Icon, Tooltip } from "@gravity-ui/uikit";

// With collapsing on, this many theses stay visible; the rest hide behind
// the curtain.
const VISIBLE_THESES = 3;

interface SummaryThesesProps {
  content: string[];
  /** Whether a long summary collapses behind the curtain; false shows it in full. */
  collapsed: boolean;
}

/**
 * A thesis list. `last` marks the list that ends the card, adding the bottom
 * margin that evens the whitespace before the divider.
 */
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

/**
 * The article summary as a thesis list. When collapsed and longer than
 * {@link VISIBLE_THESES}, the extra theses hide behind a sliding "curtain"
 * whose toggle strip sits below them and always closes the card.
 *
 * Hand-rolled rather than Gravity's Disclosure/Accordion: neither animates
 * height, nor can place the toggle below the content.
 */
export function SummaryTheses({ content, collapsed }: SummaryThesesProps) {
  const curtainId = useId();
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
          aria-controls={curtainId}
          aria-expanded={expanded}
          aria-label={label}
          onClick={() => setExpanded((value) => !value)}
          className="theses-toggle"
        >
          {/* One chevron for both states; CSS rotates it 180° on expand. */}
          <Icon data={ChevronDownWide} size={20} />
        </Button>
      </Tooltip>
    </>
  );
}
