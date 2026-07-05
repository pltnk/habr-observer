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

export function ThemeSelector({ preference, onUpdate }: ThemeSelectorProps) {
  // Until a theme is pinned the app follows the OS, so the selector shows the
  // resolved theme (the hook requires rendering inside ThemeProvider).
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
