import { Box, Text } from "@gravity-ui/uikit";

export function Header() {
  return (
    <Box as="header" className="header">
      <Text as="h1" variant="display-3">
        🧐
        <br />
        Обозреватель Хабра
      </Text>
      <Text as="h2" variant="subheader-2">
        Краткий пересказ лучших статей с Хабра от нейросети YandexGPT
      </Text>
    </Box>
  );
}
