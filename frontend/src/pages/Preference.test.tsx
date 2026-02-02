import { render, screen, waitFor, fireEvent } from "@testing-library/react";
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
  getOrders: vi.fn(),
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
    vi.mocked(api.getOrders).mockResolvedValue([]);
    
    renderPreference();
    
    // Wait for loading to complete - check that getOrders was called
    await waitFor(() => {
      expect(api.getOrders).toHaveBeenCalled();
    }, { timeout: 3000 });
    
    // Wait for form to be ready
    await waitFor(() => {
      expect(screen.getByText(/set preference/i)).toBeInTheDocument();
    }, { timeout: 3000 });

    await user.selectOptions(screen.getByRole("combobox"), "DELIVERY");
    
    // Wait for conditional fields to appear
    await waitFor(() => {
      expect(screen.getByLabelText(/address/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/pickup time/i)).toBeInTheDocument();
    });
    
    await user.type(screen.getByLabelText(/address/i), "123 Main St");
    
    // Set past date using fireEvent for datetime-local input
    const pickupInput = screen.getByLabelText(/pickup time/i) as HTMLInputElement;
    const pastDate = "2020-01-01T12:00";
    
    // Clear and set value using fireEvent (more reliable for datetime-local)
    fireEvent.change(pickupInput, { target: { value: pastDate } });
    
    // Trigger form submission
    const saveButton = screen.getByRole("button", { name: /save/i });
    await user.click(saveButton);

    // Wait for validation error to appear
    await waitFor(() => {
      const errorText = screen.queryByText(/future/i);
      expect(errorText).toBeInTheDocument();
    }, { timeout: 3000 });
    
    expect(api.createOrder).not.toHaveBeenCalled();
  });
});
