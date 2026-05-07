"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { api } from "@/lib/api";
import { setAuthToken, setSelectedTenantID } from "@/lib/auth";

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
      setSelectedTenantID(response.tenant.id);
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao criar conta.");
    } finally {
      setLoading(false);
    }
  }

  function update<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  return (
    <main className="mx-auto flex min-h-screen max-w-6xl items-center px-6 py-12 lg:px-10">
      <div className="grid w-full gap-8 lg:grid-cols-[0.95fr_1.05fr]">
        <section className="flex flex-col justify-center">
          <p className="panel-title">Onboarding SaaS</p>
          <h1 className="mt-4 text-5xl font-bold text-white">Criar workspace e entrar operando</h1>
          <p className="mt-4 max-w-xl text-lg leading-8 text-slate-300">
            Signup cria usuario, tenant, membership owner e assinatura inicial. O painel ja fica pronto para conectar o WhatsApp.
          </p>
        </section>

        <section className="panel p-8">
          <form className="grid gap-4" onSubmit={onSubmit}>
            <input className="input" placeholder="Seu nome" value={form.name} onChange={(e) => update("name", e.target.value)} required />
            <input className="input" placeholder="Email" type="email" value={form.email} onChange={(e) => update("email", e.target.value)} required />
            <input className="input" placeholder="Senha forte" type="password" value={form.password} onChange={(e) => update("password", e.target.value)} required />
            <input className="input" placeholder="Nome do workspace" value={form.tenant_name} onChange={(e) => update("tenant_name", e.target.value)} required />
            <select className="input" value={form.plan} onChange={(e) => update("plan", e.target.value)}>
              <option value="starter">Starter</option>
              <option value="growth">Growth</option>
              <option value="pro">Pro</option>
            </select>
            {error ? <p className="rounded-2xl border border-danger/30 bg-danger/10 px-4 py-3 text-sm text-danger">{error}</p> : null}
            <button className="button-primary w-full" disabled={loading} type="submit">
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
