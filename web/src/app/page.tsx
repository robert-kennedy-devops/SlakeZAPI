import Link from "next/link";

import { PLAN_OPTIONS } from "@/lib/plans";

const comparisonRows = [
  {
    label: "Mensagens incluídas",
    values: PLAN_OPTIONS.map((plan) => plan.monthlyLimit),
  },
  {
    label: "Webhooks e integrações",
    values: PLAN_OPTIONS.map((plan) => plan.webhook),
  },
  {
    label: "Perfil comercial",
    values: PLAN_OPTIONS.map((plan) => plan.idealFor),
  },
];

export default function HomePage() {
  return (
    <main className="relative overflow-hidden">
      <div className="hero-orb left-[-120px] top-10 h-72 w-72 bg-glow/20" />
      <div className="hero-orb right-[-80px] top-32 h-64 w-64 bg-neon/10" />
      <div className="grid-background absolute inset-0 opacity-30" />
      <div className="relative mx-auto flex min-h-screen max-w-7xl flex-col px-6 py-8 lg:px-10">
        <header className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <div className="surface-muted flex h-12 w-12 items-center justify-center text-lg font-bold text-white">
              S
            </div>
            <div>
              <p className="panel-title">SlakeZAPI</p>
              <h1 className="mt-1 text-xl font-bold text-white">
                WhatsApp Ops Platform
              </h1>
            </div>
          </div>
          <div className="flex gap-3">
            <Link className="button-secondary" href="/login">
              Entrar
            </Link>
            <Link className="button-primary" href="/signup">
              Criar workspace
            </Link>
          </div>
        </header>

        <section className="grid flex-1 gap-12 py-14 lg:grid-cols-[1.08fr_0.92fr] lg:items-center">
          <div>
            <p className="badge border-glow/20 bg-glow/10 text-glow">
              B2B SaaS • WhatsApp Ops • Multiworkspace
            </p>
            <h2 className="mt-6 max-w-4xl text-5xl font-bold leading-tight text-white lg:text-6xl">
              Organize atendimento, campanhas e automação de WhatsApp em uma
              oferta fácil de vender.
            </h2>
            <p className="mt-6 max-w-2xl text-lg leading-8 text-slate-300">
              A SlakeZAPI posiciona sua operação entre o improviso do WhatsApp
              manual e o custo alto das plataformas enterprise, com dashboard,
              campanhas, inbox operacional e integração em uma experiência
              pronta para comercialização.
            </p>
            <div className="mt-8 flex flex-wrap gap-4">
              <Link className="button-primary" href="/signup">
                Comecar agora
              </Link>
              <Link className="button-secondary" href="/login">
                Ver ambiente
              </Link>
            </div>
            <div className="mt-10 grid gap-4 sm:grid-cols-3">
              {[
                [
                  "Oferta clara",
                  "Planos com nomes mais comerciais e leitura imediata de valor.",
                ],
                [
                  "Preco competitivo",
                  "Faixa pensada para competir com APIs simples e ficar abaixo de suites mais caras.",
                ],
                [
                  "Operacao pronta",
                  "Conexao, campanhas, inbox, webhooks e monitoramento em um unico fluxo.",
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
          </div>

          <div className="auth-shell p-6 lg:p-7">
            <div className="absolute inset-x-10 top-0 h-px bg-gradient-to-r from-transparent via-glow/60 to-transparent" />
            <div className="grid gap-4">
              <div className="surface-muted p-5">
                <p className="section-kicker">Resumo executivo</p>
                <div className="mt-4 grid gap-4 sm:grid-cols-2">
                  <div>
                    <p className="text-3xl font-bold text-white">3 faixas</p>
                    <p className="mt-1 text-sm text-slate-400">
                      para vender conforme o estagio da operacao do cliente
                    </p>
                  </div>
                  <div>
                    <p className="text-3xl font-bold text-white">1 painel</p>
                    <p className="mt-1 text-sm text-slate-400">
                      para acompanhar vendas, suporte, automacao e governanca
                    </p>
                  </div>
                </div>
              </div>
              {[
                [
                  "Conexao e pareamento",
                  "QR, codigo de pareamento, status e retomada de instancia em tempo real.",
                ],
                [
                  "Mensageria operacional",
                  "Envios, campanhas, inbox e midias em uma jornada visual clara para a equipe.",
                ],
                [
                  "Administracao e integracao",
                  "Equipe, auditoria, webhooks, API keys e observabilidade em modulos separados.",
                ],
              ].map(([title, description]) => (
                <div
                  key={title}
                  className="rounded-3xl border border-white/10 bg-slate-950/60 p-5"
                >
                  <p className="text-base font-semibold text-white">{title}</p>
                  <p className="mt-2 text-sm leading-6 text-slate-400">
                    {description}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="pb-8">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="section-kicker">Pricing</p>
              <h3 className="mt-3 max-w-4xl text-3xl font-bold text-white lg:text-4xl">
                Um posicionamento comercial mais forte, com nomes mais premium e
                decisao de compra mais rapida.
              </h3>
              <p className="mt-3 max-w-3xl text-base leading-7 text-slate-300">
                Cada plano foi organizado para encaixar em um tipo real de
                operacao. O objetivo e facilitar venda consultiva curta, trial
                orientado e upgrade natural sem confundir o cliente.
              </p>
            </div>
            <Link className="button-primary" href="/signup">
              Escolher plano
            </Link>
          </div>

          <div className="mt-8 grid gap-5 lg:grid-cols-3">
            {PLAN_OPTIONS.map((plan) => (
              <div
                key={plan.value}
                className={`auth-shell relative flex h-full flex-col p-6 ${
                  plan.badge ? "shadow-[0_0_0_1px_rgba(87,224,194,0.18)]" : ""
                }`}
              >
                {plan.badge ? (
                  <span className="absolute right-6 top-6 rounded-full border border-glow/30 bg-glow/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-glow">
                    {plan.badge}
                  </span>
                ) : null}
                <div className="pr-24">
                  <p className="panel-title">{plan.code}</p>
                  <p className="mt-2 text-2xl font-bold text-white">
                    {plan.name}
                  </p>
                </div>
                <p className="mt-5 text-4xl font-bold text-white">
                  {plan.price}
                </p>
                <p className="mt-3 text-sm leading-6 text-slate-400">
                  {plan.summary}
                </p>
                <div className="mt-6 rounded-3xl border border-white/10 bg-slate-950/60 p-4">
                  <p className="text-xs uppercase tracking-[0.18em] text-slate-500">
                    Ideal para
                  </p>
                  <p className="mt-2 text-sm leading-6 text-slate-200">
                    {plan.idealFor}
                  </p>
                </div>
                <div className="mt-6 space-y-3">
                  {plan.highlights.map((item) => (
                    <div
                      key={item}
                      className="rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-3 text-sm text-slate-200"
                    >
                      {item}
                    </div>
                  ))}
                </div>
                <Link className="button-primary mt-6 w-full" href="/signup">
                  Escolher {plan.name}
                </Link>
              </div>
            ))}
          </div>
        </section>

        <section className="pb-16 pt-8">
          <div className="auth-shell overflow-hidden p-0">
            <div className="border-b border-white/10 px-6 py-5">
              <p className="section-kicker">Tabela comercial</p>
              <h4 className="mt-2 text-2xl font-bold text-white">
                Plano ideal para quem
              </h4>
              <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-400">
                Tabela pensada para proposta comercial, comparacao rapida e
                escolha assistida pelo time de vendas.
              </p>
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-left">
                <thead className="bg-slate-950/70">
                  <tr>
                    <th className="px-6 py-4 text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
                      Criterio
                    </th>
                    {PLAN_OPTIONS.map((plan) => (
                      <th
                        key={plan.value}
                        className="px-6 py-4 text-sm font-semibold text-white"
                      >
                        <div>{plan.name}</div>
                        <div className="mt-1 text-xs font-medium text-slate-400">
                          {plan.price}
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {comparisonRows.map((row) => (
                    <tr key={row.label} className="border-t border-white/10">
                      <td className="px-6 py-5 text-sm font-medium text-white">
                        {row.label}
                      </td>
                      {row.values.map((value, index) => (
                        <td
                          key={`${row.label}-${PLAN_OPTIONS[index].value}`}
                          className="px-6 py-5 text-sm leading-6 text-slate-300"
                        >
                          {value}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
