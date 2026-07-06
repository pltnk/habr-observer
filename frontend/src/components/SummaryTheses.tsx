import { ChevronDownWide } from "@gravity-ui/icons";
import { Button, Disclosure, Icon, Tooltip } from "@gravity-ui/uikit";

// Mirrors the original's visible_theses=3: with collapsing on, this many
// theses stay visible and the rest hide behind the disclosure.
const VISIBLE_THESES = 3;

interface SummaryThesesProps {
  content: string[];
  collapsed: boolean;
}

function ThesesList({ theses }: { theses: string[] }) {
  return (
    <ul className="theses">
      {theses.map((thesis, index) => (
        <li key={index}>{thesis}</li>
      ))}
    </ul>
  );
}

export function SummaryTheses({ content, collapsed }: SummaryThesesProps) {
  if (!collapsed || content.length <= VISIBLE_THESES) {
    return <ThesesList theses={content} />;
  }
  return (
    <>
      <ThesesList theses={content.slice(0, VISIBLE_THESES)} />
      <Disclosure className="theses-disclosure" defaultExpanded={false}>
        <Disclosure.Summary>
          {(props) => {
            const label = props.expanded
              ? "Свернуть продолжение пересказа"
              : "Развернуть продолжение пересказа";
            // The whole column-wide strip is clickable, like the original
            // Streamlit expander header.
            return (
              <Tooltip content={label}>
                <Button
                  view="flat-secondary"
                  size="l"
                  width="max"
                  id={props.id}
                  aria-controls={props.ariaControls}
                  aria-expanded={props.expanded}
                  aria-label={label}
                  onClick={props.onClick}
                  className="theses-toggle"
                >
                  {/* One chevron for both states — CSS turns it 180° on
                      expand, like Gravity's own Disclosure arrow. */}
                  <Icon data={ChevronDownWide} size={20} />
                </Button>
              </Tooltip>
            );
          }}
        </Disclosure.Summary>
        <ThesesList theses={content.slice(VISIBLE_THESES)} />
      </Disclosure>
    </>
  );
}
