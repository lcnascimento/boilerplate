import type { Decorator, Preview } from "@storybook/react-vite";

import "../src/styles/globals.css";

const withTheme: Decorator = (Story, context) => (
  <div className={context.globals.theme === "dark" ? "dark" : ""}>
    <div className="min-h-24 bg-background p-4 text-foreground">
      <Story />
    </div>
  </div>
);

const preview: Preview = {
  parameters: {
    layout: "centered",
    controls: {
      matchers: { color: /(background|color)$/i, date: /Date$/i },
    },
    // Accessibility violations fail the story tests.
    a11y: { test: "error" },
  },
  globalTypes: {
    theme: {
      description: "Color theme",
      toolbar: {
        title: "Theme",
        icon: "circlehollow",
        items: [
          { value: "light", icon: "sun", title: "Light" },
          { value: "dark", icon: "moon", title: "Dark" },
        ],
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: { theme: "light" },
  decorators: [withTheme],
  tags: ["autodocs"],
};

export default preview;
