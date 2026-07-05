import { Disclosure } from "@gravity-ui/uikit";

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
      <Disclosure
        summary="Продолжение пересказа"
        defaultExpanded={false}
        size="m"
        arrowPosition="start"
        className="theses-disclosure"
      >
        <ThesesList theses={content.slice(VISIBLE_THESES)} />
      </Disclosure>
    </>
  );
}
