"use client";
import { FormEvent, useState } from "react";
import { request, User } from "@/lib/api";

export default function AuthDialog({
  onClose,
  onSuccess,
}: {
  onClose: () => void;
  onSuccess: (token: string, user: User) => void;
}) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const form = new FormData(e.currentTarget);
    try {
      const payload: {
        email: FormDataEntryValue | null;
        password: FormDataEntryValue | null;
        displayName?: FormDataEntryValue | null;
      } = { email: form.get("email"), password: form.get("password") };
      if (mode === "register") payload.displayName = form.get("name");
      const data = await request<{ token: string; user: User }>(`/api/auth/${mode}`, {
        method: "POST",
        body: JSON.stringify(payload),
      });
      onSuccess(data.token, data.user);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="dialog-backdrop" onMouseDown={onClose}>
      <section
        className="auth-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="auth-title"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <button className="dialog-close" onClick={onClose} aria-label="Close authentication dialog">
          ×
        </button>
        <div className="crest small" aria-hidden="true">
          ♛
        </div>
        <p className="eyebrow">ROYAL MOVE</p>
        <h2 id="auth-title">{mode === "login" ? "Welcome back" : "Create an account"}</h2>
        <p className="muted">Save your analyses and keep your game history in one place.</p>
        <form onSubmit={submit}>
          {mode === "register" && (
            <label htmlFor="auth-name">
              FULL NAME
              <input
                id="auth-name"
                name="name"
                autoComplete="name"
                required
                minLength={2}
                placeholder="Your name"
              />
            </label>
          )}
          <label htmlFor="auth-email">
            EMAIL
            <input
              id="auth-email"
              name="email"
              type="email"
              autoComplete="email"
              required
              placeholder="you@example.com"
            />
          </label>
          <label htmlFor="auth-password">
            PASSWORD
            <input
              id="auth-password"
              name="password"
              type="password"
              autoComplete={mode === "login" ? "current-password" : "new-password"}
              required
              minLength={8}
              placeholder="At least 8 characters"
            />
          </label>
          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}
          <button className="primary wide" disabled={busy} aria-busy={busy}>
            {busy ? "PLEASE WAIT…" : mode === "login" ? "SIGN IN" : "CREATE ACCOUNT"}
          </button>
        </form>
        <button
          className="text-button"
          onClick={() => setMode(mode === "login" ? "register" : "login")}
        >
          {mode === "login"
            ? "New to Royal Move? Create an account"
            : "Already have an account? Sign in"}
        </button>
      </section>
    </div>
  );
}
