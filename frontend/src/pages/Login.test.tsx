import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import Login from "./Login";
import { AuthProvider } from "../context/AuthContext";
import * as api from "../api/client";

vi.mock("../api/client", () => ({ login: vi.fn(), me: vi.fn() }));

function renderLogin(initialEntries = ["/login"]) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <AuthProvider>
        <Login />
      </AuthProvider>
    </MemoryRouter>
  );
}

describe("Login", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.mocked(api.login).mockResolvedValue({ token: "fake-token" });
    vi.mocked(api.me).mockResolvedValue({ id: 1, email: "user@weel.com" });
  });

  it("shows validation for empty email", async () => {
    const user = userEvent.setup();
    renderLogin();
    await user.click(screen.getByRole("button", { name: /sign in/i }));
    await waitFor(() =>
      expect(screen.getByText(/email required/i)).toBeInTheDocument()
    );
  });
});
