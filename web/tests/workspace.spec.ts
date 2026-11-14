import { test, expect } from "@playwright/test";

test("network evidence preserves large integer IDs", async ({ page }) => {
  await page.goto("/");
  await page
    .getByRole("button", { name: "Checkout · null discount regression" })
    .first()
    .click();
  await page.waitForURL((url) => url.searchParams.has("case"));
  const id = new URL(page.url()).searchParams.get("case");
  const response = await page.request.get("/api/cases/" + id);
  const source = await response.json();
  const original = source.fixtures[0].body;
  try {
    source.fixtures[0].body = '{"id":9007199254740993,"discount":null}';
    expect(
      (await page.request.put("/api/cases/" + id, { data: source })).ok(),
    ).toBeTruthy();
    await page.reload();
    await page.locator(".network-item summary").first().click();
    await expect(page.locator(".network-item pre").first()).toContainText(
      "9007199254740993",
    );
  } finally {
    source.fixtures[0].body = original;
    await page.request.put("/api/cases/" + id, { data: source });
  }
});

test("recorded demo opens offline, verifies the fix, and roundtrips a portable case", async ({
  page,
}) => {
  const exceptions: string[] = [];
  page.on("pageerror", (e) => exceptions.push(e.message));
  await page.goto("/");
  await page
    .getByRole("button", { name: "Checkout · null discount regression" })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "Checkout · null discount regression" }),
  ).toBeVisible();
  const replay = page.frameLocator('iframe[title="Recorded browser session"]');
  await expect(
    replay.getByRole("button", { name: "Play recording" }),
  ).toBeEnabled();
  await page.screenshot({ path: "/tmp/kipuray-desktop.png", fullPage: true });
  await page.getByRole("button", { name: "Verify fix", exact: true }).click();
  await page
    .getByRole("button", { name: "Verify fix", exact: true })
    .last()
    .click();
  await expect(page.getByText("verified", { exact: true }).first()).toBeVisible(
    { timeout: 30000 },
  );
  await page.evaluate(() => scrollTo(0, 0));
  await page.screenshot({ path: "/tmp/kipuray-verified.png", fullPage: true });
  const download = page.waitForEvent("download");
  await page.getByRole("link", { name: "Export .kpr" }).click();
  const file = await download;
  await file.saveAs("/tmp/kipuray-checkout.kpr");
  await page
    .locator('input[type="file"]')
    .setInputFiles("/tmp/kipuray-checkout.kpr");
  await expect(
    page.getByRole("status").filter({ hasText: "Case imported" }),
  ).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page
      .frameLocator('iframe[title="Recorded browser session"]')
      .getByRole("button", { name: "Play recording" }),
  ).toBeEnabled();
  await page.screenshot({ path: "/tmp/kipuray-mobile.png", fullPage: true });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  expect(exceptions).toEqual([]);
});

test("response variants keep the original fixture and can be selected for execution", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("button", { name: "Checkout · null discount regression" })
    .first()
    .click();
  await page.getByRole("button", { name: "Scenario & data" }).click();
  await page.locator(".fixture-editor summary").first().click();
  await page.getByRole("button", { name: "Create a response variant" }).click();
  await page.getByLabel("Variant name").fill("Unavailable API");
  await page.getByLabel("Status", { exact: true }).fill("500");
  await page.getByLabel("Replacement JSON").fill('{"message":"Try again"}');
  await page.getByRole("button", { name: "Add variant", exact: true }).click();
  await page.getByRole("button", { name: "Save case", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Saved", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Verify fix", exact: true }).click();
  await expect(
    page.getByRole("combobox", { name: "Scenario", exact: true }),
  ).toHaveValue(/.+/);
  await expect(
    page
      .getByRole("combobox", { name: "Scenario", exact: true })
      .getByRole("option", { name: "Original responses" }),
  ).toHaveCount(1);
});

test("shared cases are read-only and revocable", async ({ page, browser }) => {
  await page.goto("/");
  await page
    .getByRole("button", { name: "Checkout · null discount regression" })
    .first()
    .click();
  await page.waitForURL((url) => url.searchParams.has("case"));
  const id = new URL(page.url()).searchParams.get("case");
  const created = await page.request.post("/api/cases/" + id + "/share", {
    data: {},
  });
  expect(created.ok()).toBeTruthy();
  const link = (await created.json()).url;
  const guest = await browser.newContext();
  const viewer = await guest.newPage();
  await viewer.goto("http://127.0.0.1:4317" + link);
  await expect(
    viewer.getByRole("heading", {
      name: "Checkout · null discount regression",
    }),
  ).toBeVisible();
  await expect(
    viewer.getByRole("button", { name: "New recording" }),
  ).toHaveCount(0);
  await expect(
    viewer.getByRole("button", { name: "Share", exact: true }),
  ).toHaveCount(0);
  await page.request.delete("/api/cases/" + id + "/share");
  await viewer.reload();
  await expect(viewer.getByRole("alert")).toBeVisible();
  await guest.close();
});

test("a manual capture is checkpointed and survives reopening the workspace", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "New recording" }).click();
  const setup = page.getByRole("dialog");
  await expect(setup).toBeVisible();
  await setup.getByRole("button", { name: "Open recording window" }).click();
  await expect(
    page.getByRole("button", { name: "Stop & review" }),
  ).toBeVisible();
  await expect
    .poll(
      async () => {
        const response = await page.request.get("/api/cases");
        return (await response.json()).some((c: { title: string }) =>
          c.title.startsWith("Recording in progress"),
        );
      },
      { timeout: 10000 },
    )
    .toBeTruthy();
  await page.getByRole("button", { name: "Stop & review" }).click();
  await expect(
    page.getByRole("heading", { name: /Recording ·/ }),
  ).toBeVisible();
  await page.waitForURL((url) => url.searchParams.has("case"));
  await page.reload();
  await expect(
    page.getByRole("heading", { name: /Recording ·/ }),
  ).toBeVisible();
  await expect(
    page
      .frameLocator('iframe[title="Recorded browser session"]')
      .getByRole("button", { name: "Play recording" }),
  ).toBeEnabled();
});
