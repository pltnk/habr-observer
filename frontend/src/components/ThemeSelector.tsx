import { Moon, Sun } from "@gravity-ui/icons";
import {
  Icon,
  SegmentedRadioGroup,
  Tooltip,
  useThemeType,
} from "@gravity-ui/uikit";

import type { ThemePreference } from "../hooks/useThemePreference";

interface ThemeSelectorProps {
  preference: ThemePreference | null;
  onUpdate: (value: ThemePreference) => void;
}

/**
 * Segmented control pinning the theme to light or dark. Until pinned, the app
 * follows the OS and the control shows the resolved theme.
 */
export function ThemeSelector({ preference, onUpdate }: ThemeSelectorProps) {
  // Resolved active theme for the unpinned case; requires a ThemeProvider ancestor.
  const resolvedTheme = useThemeType();
  return (
    <Tooltip content="Светлая или тёмная тема">
      <SegmentedRadioGroup<ThemePreference>
        size="m"
        aria-label="Тема"
        value={preference ?? resolvedTheme}
        onUpdate={onUpdate}
      >
        {/* Icon-only options: the aria-labels carry the names. */}
        <SegmentedRadioGroup.Option
          value="light"
          controlProps={{ "aria-label": "Светлая" }}
        >
          <Icon data={Sun} size={16} />
        </SegmentedRadioGroup.Option>
        <SegmentedRadioGroup.Option
          value="dark"
          controlProps={{ "aria-label": "Тёмная" }}
        >
          <Icon data={Moon} size={16} />
        </SegmentedRadioGroup.Option>
      </SegmentedRadioGroup>
    </Tooltip>
  );
}
