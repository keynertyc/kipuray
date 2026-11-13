import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: "http://127.0.0.1:4317",
    channel: "chrome",
    headless: true,
    trace: "retain-on-failure",
    video: process.env.KIPURAY_RECORD_VIDEO
      ? { mode: "on", size: { width: 1280, height: 800 } }
      : "off",
  },
  webServer: {
    command:
      "../bin/kipuray demo --no-open --headless --data-dir /tmp/kipuray-playwright",
    url: "http://127.0.0.1:4317/healthz",
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
  },
  reporter: "list",
});
