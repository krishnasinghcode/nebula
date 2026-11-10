import { expect, test } from "@playwright/test";
import { existsSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

function loadToken(): string {
  if (process.env.ORBIT_TOKEN) return process.env.ORBIT_TOKEN;
  const p = join(homedir(), ".orbit", "token");
  return existsSync(p) ? readFileSync(p, "utf8").trim() : "";
}

// The dashboard proxies /v1 to orbit serve; token from the serve host.
const token = loadToken();
test.skip(!token, "no Orbit token available (set ORBIT_TOKEN or run on a host with ~/.orbit/token)");

test("create → connect info → terminate", async ({ page }) => {
  await page.addInitScript((t) => localStorage.setItem("orbit-token", t), token);
  await page.goto("/");
  await expect(page.getByTestId("create-wizard")).toBeVisible();

  const name = `dash-${Date.now() % 100000}`;
  await page.getByTestId("input-name").fill(name);
  await page.getByTestId("input-image").selectOption("ubuntu-24.04");
  await page.getByTestId("input-type").selectOption("orbit.micro");
  await page.getByTestId("trust-ack").click();
  await page.getByTestId("create-submit").click();

  await expect(page.getByTestId(`row-${name}`)).toBeVisible({ timeout: 30_000 });

  await page.getByTestId(`row-${name}`).click();
  await expect(page.getByTestId("instance-detail")).toBeVisible();
  // Wait until actually running before terminating (409 otherwise).
  await expect(page.getByTestId("instance-detail")).toContainText("running", { timeout: 60_000 });

  page.on("dialog", (d) => d.accept());
  await page.getByTestId("btn-terminate").click();
  // The detail view keeps polling; terminated instances stay listed.
  await expect(page.getByTestId("instance-detail")).toContainText("terminated", { timeout: 60_000 });
});
