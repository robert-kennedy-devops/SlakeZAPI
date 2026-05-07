import { expect, test, type Page, type Route } from "@playwright/test";

const sessionPayload = {
  token: "app-token-123",
  expires_at: "2030-01-01T00:15:00Z",
  refresh_expires_at: "2030-01-08T00:00:00Z",
  user: {
    id: "user-1",
    email: "owner@example.com",
    name: "Owner Example",
    active: true,
    created_at: "2030-01-01T00:00:00Z",
  },
  tenant: {
    id: "tenant-1",
    name: "Workspace Prime",
    email: "owner@example.com",
    active: true,
    created_at: "2030-01-01T00:00:00Z",
  },
  membership: {
    id: "membership-1",
    tenant_id: "tenant-1",
    user_id: "user-1",
    role: "owner",
    created_at: "2030-01-01T00:00:00Z",
  },
};

async function mockRealtime(page: Page) {
  await page.addInitScript(() => {
    class MockWebSocket {
      url: string;
      readyState = 1;
      onmessage: ((event: MessageEvent<string>) => void) | null = null;
      onopen: (() => void) | null = null;
      onclose: (() => void) | null = null;

      constructor(url: string) {
        this.url = url;
        queueMicrotask(() => this.onopen?.());
      }

      close() {
        this.readyState = 3;
        this.onclose?.();
      }

      send() {}
      addEventListener() {}
      removeEventListener() {}
    }

    // @ts-expect-error test shim
    window.WebSocket = MockWebSocket;
  });
}

async function fulfillJSON(
  route: Route,
  body: unknown,
  extraHeaders?: Record<string, string>,
) {
  await route.fulfill({
    status: 200,
    headers: {
      "Content-Type": "application/json",
      ...(extraHeaders ?? {}),
    },
    body: JSON.stringify(body),
  });
}

async function mockDashboardAPI(page: Page) {
  await page.route("**/app/auth/me**", async (route) => {
    await fulfillJSON(route, {
      user: sessionPayload.user,
      tenant: sessionPayload.tenant,
      membership: sessionPayload.membership,
      memberships: [sessionPayload.membership],
    });
  });
  await page.route("**/app/tenant/summary**", async (route) => {
    await fulfillJSON(route, {
      tenant: sessionPayload.tenant,
      session: {
        tenant_id: "tenant-1",
        status: "connected",
        phone: "5511999999999",
        updated_at: "2030-01-01T00:10:00Z",
      },
      usage: {
        tenant_id: "tenant-1",
        month: "2030-01",
        sent: 12,
        received: 3,
        updated_at: "2030-01-01T00:10:00Z",
      },
      plan: {
        id: "sub-1",
        tenant_id: "tenant-1",
        plan_id: "plan_growth",
        status: "active",
        period_end: "2030-02-01T00:00:00Z",
        created_at: "2030-01-01T00:00:00Z",
      },
    });
  });
  await page.route("**/app/whatsapp/status**", async (route) => {
    await fulfillJSON(route, {
      tenant_id: "tenant-1",
      status: "connected",
      phone: "5511999999999",
      updated_at: "2030-01-01T00:10:00Z",
    });
  });
  await page.route("**/app/messages", async (route) => {
    if (route.request().method() === "POST") {
      await fulfillJSON(route, {
        message_id: "msg-1",
        status: "sent",
      });
      return;
    }
    await fulfillJSON(route, [
      {
        id: "msg-1",
        tenant_id: "tenant-1",
        whatsapp_id: "wa-1",
        phone: "5511999999999",
        body: "Mensagem inicial",
        type: "text",
        direction: "outbound",
        status: "sent",
        sent_at: "2030-01-01T00:05:00Z",
        created_at: "2030-01-01T00:05:00Z",
      },
    ]);
  });
  await page.route("**/app/messages/send", async (route) => {
    const payload = route.request().postDataJSON() as {
      phone?: string;
      message?: string;
    };
    expect(payload.phone).toBe("5511999999999");
    expect(payload.message).toBe("Teste Playwright");
    await fulfillJSON(route, { message_id: "msg-2", status: "sent" });
  });
  await page.route("**/app/webhooks**", async (route) => {
    await fulfillJSON(route, []);
  });
  await page.route("**/app/apikeys**", async (route) => {
    await fulfillJSON(route, []);
  });
  await page.route("**/app/usage**", async (route) => {
    await fulfillJSON(route, {
      tenant_id: "tenant-1",
      month: "2030-01",
      sent: 12,
      received: 3,
      updated_at: "2030-01-01T00:10:00Z",
    });
  });
  await page.route("**/app/members**", async (route) => {
    await fulfillJSON(route, [
      {
        id: "membership-1",
        tenant_id: "tenant-1",
        user_id: "user-1",
        email: "owner@example.com",
        name: "Owner Example",
        role: "owner",
        active: true,
        created_at: "2030-01-01T00:00:00Z",
      },
    ]);
  });
}

test("signup redirects to dashboard with workspace context", async ({
  page,
}) => {
  await mockRealtime(page);
  await page.route("**/app/auth/signup", async (route) => {
    await fulfillJSON(route, sessionPayload, {
      "Set-Cookie": "slakezapi_rt=fake-refresh; Path=/; HttpOnly",
    });
  });
  await mockDashboardAPI(page);

  await page.goto("/signup");
  await page.getByTestId("signup-name").fill("Owner Example");
  await page.getByTestId("signup-email").fill("owner@example.com");
  await page.getByTestId("signup-password").fill("supersecret123");
  await page.getByTestId("signup-tenant").fill("Workspace Prime");
  await page.getByTestId("signup-submit").click();

  await page.waitForURL("**/dashboard");
  await expect(page.getByTestId("dashboard-root")).toBeVisible();
  await expect(page.getByText("Workspace Prime")).toBeVisible();
});

test("login loads dashboard and sends a message", async ({ page }) => {
  await mockRealtime(page);
  await page.route("**/app/auth/login", async (route) => {
    await fulfillJSON(route, sessionPayload, {
      "Set-Cookie": "slakezapi_rt=fake-refresh; Path=/; HttpOnly",
    });
  });
  await mockDashboardAPI(page);

  await page.goto("/login");
  await page.getByTestId("login-email").fill("owner@example.com");
  await page.getByTestId("login-password").fill("supersecret123");
  await page.getByTestId("login-submit").click();

  await page.waitForURL("**/dashboard");
  await expect(page.getByTestId("dashboard-root")).toBeVisible();
  await expect(page.getByTestId("member-email")).toBeVisible();

  await page.getByTestId("send-phone").fill("5511999999999");
  await page.getByTestId("send-message").fill("Teste Playwright");
  await page.getByTestId("send-submit").click();

  await expect(page.getByText("Mensagem enviada.")).toBeVisible();
});
