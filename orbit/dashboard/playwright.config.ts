import { defineConfig } from "@playwright/test";

// Smoke tests run against a real `orbit serve` (default http://127.0.0.1:7420).
export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL: process.env.ORBIT_URL ?? "http://127.0.0.1:5173",
  },
  webServer: process.env.ORBIT_URL
    ? undefined
    : {
        // Vite dev server proxies /v1 to orbit serve (127.0.0.1:7420).
        command: "npm run dev",
        url: "http://127.0.0.1:5173",
        reuseExistingServer: true,
        timeout: 60_000,
      },
});
