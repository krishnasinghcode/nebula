import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: "127.0.0.1",
    proxy: {
      // Dev convenience only; orbit serves on 127.0.0.1:7420.
      "/v1": "http://127.0.0.1:7420",
    },
  },
  test: {
    environment: "jsdom",
    setupFiles: "./src/test/setup.ts",
    globals: true,
    exclude: ["**/node_modules/**", "**/dist/**", "e2e/**"],
  },
});
