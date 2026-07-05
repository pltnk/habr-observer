import { Flex, HelpMark, Switch } from "@gravity-ui/uikit";

interface SummaryToggleProps {
  checked: boolean;
  onUpdate: (checked: boolean) => void;
}

export function SummaryToggle({ checked, onUpdate }: SummaryToggleProps) {
  return (
    <Flex
      justifyContent="center"
      alignItems="center"
      gap={2}
      className="summary-toggle"
    >
      <Switch
        size="m"
        checked={checked}
        onUpdate={onUpdate}
        content="Сворачивать пересказы"
      />
      <HelpMark>
        Отключите, чтобы показывать пересказы целиком, не сворачивая их
      </HelpMark>
    </Flex>
  );
}
