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
import { PLAN_OPTIONS } from "@/lib/plans";

export default function SignUpPage() {
  const router = useRouter();
  const [form, setForm] = useState({
    name: "",
    email: "",
    password: "",
    tenant_name: "",
    plan: "trial",
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
            Criar workspace com oferta comercial mais forte
          </h1>
          <p className="mt-4 max-w-xl text-lg leading-8 text-slate-300">
            O onboarding precisa reduzir friccao e comunicar valor. Escolha um
            plano competitivo, comece com uma degustacao gratuita de 2 dias e
            entre direto na operacao.
          </p>
          <div className="mt-8 grid gap-3">
            {[
              "Planos com precificacao simples para pequenas e medias operacoes.",
              "Nomes mais premium para vender melhor e explicar menos.",
              "Degustacao completa por 48 horas antes da ativacao comercial.",
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
            <div className="grid gap-3" data-testid="signup-plan">
              {PLAN_OPTIONS.map((plan) => {
                const selected = form.plan === plan.value;
                return (
                  <button
                    key={plan.value}
                    className={`rounded-3xl border p-4 text-left transition ${
                      selected
                        ? "border-glow bg-glow/10 shadow-[0_0_0_1px_rgba(87,224,194,0.25)]"
                        : "border-white/10 bg-slate-950/50 hover:border-white/20"
                    }`}
                    onClick={() => update("plan", plan.value)}
                    type="button"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="text-base font-semibold text-white">
                          {plan.name}
                        </p>
                        <p className="mt-1 text-sm text-slate-300">
                          {plan.price}
                        </p>
                      </div>
                      <span
                        className={`rounded-full px-3 py-1 text-xs font-medium ${
                          selected
                            ? "bg-glow/20 text-glow"
                            : "bg-white/5 text-slate-400"
                        }`}
                      >
                        {selected ? "Selecionado" : "Escolher"}
                      </span>
                    </div>
                    <p className="mt-3 text-sm leading-6 text-slate-400">
                      {plan.summary}
                    </p>
                    <p className="mt-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                      {plan.details}
                    </p>
                    <p className="mt-2 text-xs leading-5 text-slate-500">
                      Ideal para: {plan.idealFor}
                    </p>
                  </button>
                );
              })}
            </div>
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
