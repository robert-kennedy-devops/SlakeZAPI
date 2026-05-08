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

export default function SignUpPage() {
  const router = useRouter();
  const [form, setForm] = useState({
    name: "",
    email: "",
    password: "",
    tenant_name: "",
    plan: "starter",
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");
    try {
      const response = await api.signUp(form);
      setAuthToken(response.token);
      setAuthTokenExpiry(response.expires_at);
      setSelectedTenantID(response.tenant.id);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao criar conta.");
    } finally {
      setLoading(false);
    }
  }

  function update<K extends keyof typeof form>(
    key: K,
    value: (typeof form)[K],
  ) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  return (
    <main className="relative mx-auto flex min-h-screen max-w-6xl items-center px-6 py-12 lg:px-10">
      <div className="hero-orb right-[-60px] top-20 h-56 w-56 bg-neon/15" />
      <div className="grid w-full gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <section className="flex flex-col justify-center">
          <p className="section-kicker">Onboarding SaaS</p>
          <h1 className="mt-4 text-5xl font-bold text-white">
            Criar workspace com cara de produto pronto
          </h1>
          <p className="mt-4 max-w-xl text-lg leading-8 text-slate-300">
            O primeiro fluxo precisa vender confianca. Este onboarding prepara
            usuario, workspace, permissao e plano inicial para o cliente entrar
            direto na operacao do WhatsApp.
          </p>
          <div className="mt-8 grid gap-3">
            {[
              "Cadastro guiado para reduzir friccao de ativacao.",
              "Plano inicial claro para facilitar demonstracao comercial.",
              "Workspace pronto para conectar a primeira instancia.",
            ].map((item) => (
              <div
                key={item}
                className="surface-muted px-4 py-3 text-sm text-slate-300"
              >
                {item}
              </div>
            ))}
          </div>
        </section>

        <section className="auth-shell p-8">
          <form
            className="grid gap-4"
            data-testid="signup-form"
            onSubmit={onSubmit}
          >
            <div>
              <p className="text-lg font-semibold text-white">
                Criar workspace
              </p>
              <p className="mt-1 text-sm text-slate-400">
                Configure a conta principal e deixe o ambiente pronto para
                demonstracao.
              </p>
            </div>
            <input
              className="input"
              data-testid="signup-name"
              placeholder="Seu nome"
              value={form.name}
              onChange={(e) => update("name", e.target.value)}
              required
            />
            <input
              className="input"
              data-testid="signup-email"
              placeholder="Email"
              type="email"
              value={form.email}
              onChange={(e) => update("email", e.target.value)}
              required
            />
            <input
              className="input"
              data-testid="signup-password"
              placeholder="Senha forte"
              type="password"
              value={form.password}
              onChange={(e) => update("password", e.target.value)}
              required
            />
            <input
              className="input"
              data-testid="signup-tenant"
              placeholder="Nome do workspace"
              value={form.tenant_name}
              onChange={(e) => update("tenant_name", e.target.value)}
              required
            />
            <select
              className="input"
              data-testid="signup-plan"
              value={form.plan}
              onChange={(e) => update("plan", e.target.value)}
            >
              <option value="starter">Starter</option>
              <option value="growth">Growth</option>
              <option value="pro">Pro</option>
            </select>
            {error ? (
              <p className="rounded-2xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">
                {error}
              </p>
            ) : null}
            <button
              className="button-primary w-full"
              data-testid="signup-submit"
              disabled={loading}
              type="submit"
            >
              {loading ? "Criando..." : "Criar workspace"}
            </button>
          </form>
          <p className="mt-6 text-sm text-slate-400">
            Ja tem conta?{" "}
            <Link className="text-glow hover:underline" href="/login">
              Entrar
            </Link>
          </p>
        </section>
      </div>
    </main>
  );
}
