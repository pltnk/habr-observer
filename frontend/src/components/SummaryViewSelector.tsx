import {
  ChevronsCollapseVertical,
  ChevronsExpandVertical,
} from "@gravity-ui/icons";
import { Icon, SegmentedRadioGroup, Tooltip } from "@gravity-ui/uikit";

type SummaryView = "collapsed" | "expanded";

interface SummaryViewSelectorProps {
  collapsed: boolean;
  onUpdate: (collapsed: boolean) => void;
}

/**
 * Segmented control toggling article summaries between collapsed («Кратко»)
 * and full («Целиком»).
 */
export function SummaryViewSelector({
  collapsed,
  onUpdate,
}: SummaryViewSelectorProps) {
  return (
    <Tooltip content="Показывать пересказы кратко или целиком">
      <SegmentedRadioGroup<SummaryView>
        size="m"
        aria-label="Вид пересказов"
        value={collapsed ? "collapsed" : "expanded"}
        onUpdate={(value) => onUpdate(value === "collapsed")}
      >
        {/* Icon-only options: the aria-labels carry the names. */}
        <SegmentedRadioGroup.Option
          value="collapsed"
          controlProps={{ "aria-label": "Кратко" }}
        >
          <Icon data={ChevronsCollapseVertical} size={16} />
        </SegmentedRadioGroup.Option>
        <SegmentedRadioGroup.Option
          value="expanded"
          controlProps={{ "aria-label": "Целиком" }}
        >
          <Icon data={ChevronsExpandVertical} size={16} />
        </SegmentedRadioGroup.Option>
      </SegmentedRadioGroup>
    </Tooltip>
  );
}
