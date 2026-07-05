import {
  ChevronsCollapseVertical,
  ChevronsExpandVertical,
} from "@gravity-ui/icons";
import { Flex, Icon, SegmentedRadioGroup, Tooltip } from "@gravity-ui/uikit";

type SummaryView = "collapsed" | "expanded";

interface SummaryViewSelectorProps {
  collapsed: boolean;
  onUpdate: (collapsed: boolean) => void;
}

// Post-parity product change: the migration's import allowlist excluded
// SegmentedRadioGroup/Icon/Tooltip/@gravity-ui/icons for Streamlit parity;
// this deliberately replaces the parity-era Switch with a persisted view mode.
export function SummaryViewSelector({
  collapsed,
  onUpdate,
}: SummaryViewSelectorProps) {
  return (
    <Flex justifyContent="center" className="summary-view-selector">
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
    </Flex>
  );
}
