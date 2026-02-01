import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Preference from "./Preference";
import * as api from "../api/client";
import { AuthProvider } from "../context/AuthContext";

vi.mock("../api/client", () => ({
  me: vi.fn(),
  getOrder: vi.fn(),
  getOrderSummary: vi.fn(),
  createOrder: vi.fn(),
  updateOrder: vi.fn(),
}));

function renderPreferenceWithOrder(orderId?: string) {
  if (orderId) localStorage.setItem("orderId", orderId);
  return render(
    <MemoryRouter>
      <AuthProvider>
        <Preference />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe("Summary (on Preference page)", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("token", "fake-token");
    vi.mocked(api.me).mockResolvedValue({ id: 1, email: "user@weel.com" });
  });

  it("reflects backend order data when loaded", async () => {
    vi.mocked(api.getOrder).mockResolvedValue({
      id: 42,
      user_id: 1,
      preference: "DELIVERY",
      address: "123 Main St",
      pickup_time: "2030-06-01T12:00:00Z",
      created_at: "2025-01-01T00:00:00Z",
    });

    renderPreferenceWithOrder("42");
    await waitFor(() => {
      expect(api.getOrder).toHaveBeenCalledWith(42);
    });
    expect(screen.getByText(/42/)).toBeInTheDocument();
    expect(screen.getByText(/DELIVERY/)).toBeInTheDocument();
    expect(screen.getByText(/123 Main St/)).toBeInTheDocument();
  });

  it("calls getOrderSummary and shows AI summary when Generate AI summary is clicked", async () => {
    const user = userEvent.setup();
    vi.mocked(api.getOrder).mockResolvedValue({
      id: 1,
      user_id: 1,
      preference: "IN_STORE",
      created_at: "2025-01-01T00:00:00Z",
    });
    vi.mocked(api.getOrderSummary).mockResolvedValue({
      summary: "Your in-store order #1 is ready for pickup.",
      source: "ai",
    });

    renderPreferenceWithOrder("1");
    await waitFor(() => {
      expect(api.getOrder).toHaveBeenCalledWith(1);
    });

    const btn = screen.getByRole("button", { name: /generate ai summary/i });
    await user.click(btn);

    await waitFor(() => {
      expect(api.getOrderSummary).toHaveBeenCalledWith(1);
    });
    await waitFor(() => {
      expect(
        screen.getByText(/Your in-store order #1 is ready for pickup./)
      ).toBeInTheDocument();
    });
    expect(screen.getByText(/Generated with AI/)).toBeInTheDocument();
  });
});
