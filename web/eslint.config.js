import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["**/dist/**", "**/node_modules/**", "shared/src/api/*.d.ts", "**/coverage/**"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  { files: ["e2e/**/*.mjs", "*.config.{js,ts}"], languageOptions: { globals: { ...globals.node } } },
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: { globals: { ...globals.browser } },
    plugins: { "react-hooks": reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_" }],
      // Physical directions break RTL layouts: use logical properties (doc 09 §3).
      "no-restricted-syntax": ["error", {
        selector: "Property[key.name=/^(marginLeft|marginRight|paddingLeft|paddingRight|left|right|textAlign)$/]",
        message: "Use logical properties (marginInlineStart/End, insetInlineStart/End, textAlign: 'start'/'end') for RTL."
      }]
    }
  }
);
