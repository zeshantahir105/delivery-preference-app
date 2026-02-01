import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Preference from "./Preference";
import * as api from "../api/client";
import { AuthProvider } from "../context/AuthContext";

vi.mock("../api/client", () => ({
  me: vi.fn(),
  createOrder: vi.fn(),
  getOrder: vi.fn(),
  getOrderSummary: vi.fn(),
  updateOrder: vi.fn(),
}));

function renderPreference() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <Preference />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe("Preference", () => {
  beforeEach(() => {
    localStorage.clear();
    localStorage.setItem("token", "fake-token");
    vi.mocked(api.me).mockResolvedValue({ id: 1, email: "user@weel.com" });
  });

  it("rejects past datetime for DELIVERY", async () => {
    const user = userEvent.setup();
    renderPreference();
    await screen.findByText(/delivery preference/i);

    await user.selectOptions(screen.getByRole("combobox"), "DELIVERY");
    await user.type(screen.getByLabelText(/address/i), "123 Main St");
    const pastDate = "2020-01-01T12:00";
    const pickupInput = screen.getByLabelText(/pickup time/i);
    await user.type(pickupInput, pastDate);
    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(screen.getByText(/future/i)).toBeInTheDocument();
    });
    expect(api.createOrder).not.toHaveBeenCalled();
  });
});
