import { useState } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { login } from "../api/client";
import { useAuth } from "../context/AuthContext";

const schema = z.object({
  email: z.string().min(1, "Email required").email("Invalid email"),
  password: z.string().min(1, "Password required"),
});

type FormData = z.infer<typeof schema>;

export default function Login() {
  const { setToken, user } = useAuth();
  const navigate = useNavigate();
  const [submitError, setSubmitError] = useState("");

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  if (user) {
    navigate("/", { replace: true });
    return null;
  }

  async function onSubmit(data: FormData) {
    setSubmitError("");
    try {
      const { token } = await login(data.email, data.password);
      setToken(token);
      navigate("/", { replace: true });
    } catch (e) {
      setSubmitError(e instanceof Error ? e.message : "Login failed");
    }
  }

  return (
    <div className="page page--login">
      <div className="login-card-wrapper">
        <img src="/favicon.svg" alt="" className="login-logo" aria-hidden />
        {/* <h1 className="page-title page--login-title">Sign in</h1> */}
        <p className="page--login-subtitle">
          Enter your credentials to access the app.
        </p>
        <div className="card login-card">
          <form onSubmit={handleSubmit(onSubmit)} className="login-form">
            <div className="form-group">
              <label htmlFor="email" className="label">
                Email
              </label>
              <input
                id="email"
                type="email"
                className="input"
                placeholder="you@example.com"
                autoComplete="email"
                {...register("email")}
              />
              {errors.email && <p className="error">{errors.email.message}</p>}
            </div>
            <div className="form-group">
              <label htmlFor="password" className="label">
                Password
              </label>
              <input
                id="password"
                type="password"
                className="input"
                placeholder="••••••••"
                autoComplete="current-password"
                {...register("password")}
              />
              {errors.password && (
                <p className="error">{errors.password.message}</p>
              )}
            </div>
            {submitError && <p className="error">{submitError}</p>}
            <button
              type="submit"
              disabled={isSubmitting}
              className="btn btn-primary login-submit"
            >
              {isSubmitting ? "Signing in…" : "Sign in"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
