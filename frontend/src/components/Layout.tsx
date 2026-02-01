import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

const APP_NAME = "Delivery Preference App";
const FOOTER_AUTHOR = "Muhammad Zeshan Tahir";
const FOOTER_EMAIL = "zeshantahir105@gmail.com";

export default function Layout() {
  const { user, signOut } = useAuth();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const isFullWidthPage = pathname === "/" || pathname === "/summary";
  const isLoginPage = pathname === "/login";

  function handleLogout() {
    signOut();
    navigate("/login", { replace: true });
  }

  return (
    <div className="app-layout">
      {!isLoginPage && (
        <header className="navbar">
          <div className="navbar-inner">
            <div className="navbar-brand">
              <img src="/favicon.svg" alt="" className="navbar-logo" aria-hidden />
              <h1 className="navbar-brand-title">{APP_NAME}</h1>
            </div>
            {user && (
              <div className="navbar-user">
                <div className="navbar-avatar" aria-hidden>
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  >
                    <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
                    <circle cx="12" cy="7" r="4" />
                  </svg>
                </div>
                <span className="navbar-email">{user.email}</span>
                <button
                  type="button"
                  onClick={handleLogout}
                  className="btn btn-nav-logout"
                  aria-label="Log out"
                >
                  Logout
                </button>
              </div>
            )}
          </div>
        </header>
      )}

      <main className={`app-main ${isLoginPage ? "app-main--login" : ""} ${isFullWidthPage ? "app-main--full" : ""}`}>
        <div className={`app-main-inner ${isFullWidthPage ? "app-main-inner--full" : ""}`}>
          <Outlet />
        </div>
      </main>

      {!isLoginPage && (
        <footer className="app-footer">
          <span>{FOOTER_AUTHOR} | </span>
          <a href={`mailto:${FOOTER_EMAIL}`} className="app-footer-link">
            {FOOTER_EMAIL}
          </a>
        </footer>
      )}
    </div>
  );
}
