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
    <main className="relative mx-auto flex min-h-screen max-w-6xl items-center px-6 py-12 lg:px-10">
      <div className="hero-orb left-[-80px] top-20 h-56 w-56 bg-glow/20" />
      <div className="grid w-full gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <section className="flex flex-col justify-center">
          <p className="section-kicker">Acesso ao workspace</p>
          <h1 className="mt-4 text-5xl font-bold text-white">
            Entrar e retomar a operacao em segundos
          </h1>
          <p className="mt-4 max-w-xl text-lg leading-8 text-slate-300">
            Seu cliente precisa sentir controle logo no login. Esta entrada
            prioriza clareza, continuidade de sessao e foco no que importa:
            conexao, mensagens e performance do workspace.
          </p>
          <div className="mt-8 grid gap-3 sm:grid-cols-2">
            {[
              [
                "Operacao centralizada",
                "Instancias, inbox e campanhas no mesmo lugar.",
              ],
              [
                "Acesso por papel",
                "Owner, admin, operator e viewer com governanca clara.",
              ],
            ].map(([title, description]) => (
              <div key={title} className="surface-muted p-4">
                <p className="text-sm font-semibold text-white">{title}</p>
                <p className="mt-2 text-sm leading-6 text-slate-400">
                  {description}
                </p>
              </div>
            ))}
          </div>
        </section>

        <section className="auth-shell p-8">
          <form
            className="space-y-4"
            data-testid="login-form"
            onSubmit={onSubmit}
          >
            <div>
              <p className="text-lg font-semibold text-white">
                Entrar no painel
              </p>
              <p className="mt-1 text-sm text-slate-400">
                Use suas credenciais para abrir o workspace ativo.
              </p>
            </div>
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
