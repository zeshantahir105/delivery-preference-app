import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router-dom";
import App from "./App";
import { AuthProvider } from "./context/AuthContext";

vi.mock("./api/client", () => ({ me: vi.fn() }));

describe("App auth guard", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("redirects unauthenticated user to login when visiting /", async () => {
    const { me } = await import("./api/client");
    vi.mocked(me).mockRejectedValue(new Error("unauthorized"));
    render(
      <MemoryRouter initialEntries={["/"]}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>
    );
    await screen.findByRole("heading", { name: /sign in/i });
    expect(screen.getByRole("heading", { name: /sign in/i })).toBeInTheDocument();
  });
});
