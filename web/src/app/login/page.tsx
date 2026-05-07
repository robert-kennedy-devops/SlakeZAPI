"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { api } from "@/lib/api";
import {
  setAuthToken,
  setAuthTokenExpiry,
  setSelectedTenantID,
} from "@/lib/auth";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await api.login({ email, password });
      setAuthToken(response.token);
      setAuthTokenExpiry(response.expires_at);
      setSelectedTenantID(response.tenant.id);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao autenticar.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-6xl items-center px-6 py-12 lg:px-10">
      <div className="grid w-full gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <section className="flex flex-col justify-center">
          <p className="panel-title">Acesso ao Console</p>
          <h1 className="mt-4 text-5xl font-bold text-white">
            Entrar no workspace
          </h1>
          <p className="mt-4 max-w-xl text-lg leading-8 text-slate-300">
            Use o login da camada app para operar tenants, sessoes WhatsApp,
            mensagens e webhooks sem depender de API key manual.
          </p>
        </section>

        <section className="panel p-8">
          <form
            className="space-y-4"
            data-testid="login-form"
            onSubmit={onSubmit}
          >
            <div>
              <label className="mb-2 block text-sm text-slate-300">Email</label>
              <input
                className="input"
                data-testid="login-email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                type="email"
                required
              />
            </div>
            <div>
              <label className="mb-2 block text-sm text-slate-300">Senha</label>
              <input
                className="input"
                data-testid="login-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                type="password"
                required
              />
            </div>
            {error ? (
              <p className="rounded-2xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {error}
              </p>
            ) : null}
            <button
              className="button-primary w-full"
              data-testid="login-submit"
              disabled={loading}
              type="submit"
            >
              {loading ? "Entrando..." : "Entrar"}
            </button>
          </form>
          <p className="mt-6 text-sm text-slate-400">
            Ainda nao tem workspace?{" "}
            <Link className="text-glow hover:underline" href="/signup">
              Criar conta
            </Link>
          </p>
        </section>
      </div>
    </main>
  );
}
