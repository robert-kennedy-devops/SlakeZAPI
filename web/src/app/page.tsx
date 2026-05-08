import Link from "next/link";

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
              B2B SaaS • Operacao assistida • Multiworkspace
            </p>
            <h2 className="mt-6 max-w-4xl text-5xl font-bold leading-tight text-white lg:text-6xl">
              Venda um produto que parece plataforma, nao painel interno.
            </h2>
            <p className="mt-6 max-w-2xl text-lg leading-8 text-slate-300">
              Centralize onboarding, conexao de instancias, mensageria,
              campanhas, inbox operacional e webhooks em uma experiencia
              organizada para o cliente entender valor nos primeiros minutos.
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
                  "Onboarding claro",
                  "Cadastro, workspace e pareamento com narrativa guiada.",
                ],
                [
                  "Operacao visivel",
                  "Status, uso, mensagens e alertas em leitura instantanea.",
                ],
                [
                  "Recursos avancados",
                  "Automacoes, webhooks e equipe sem poluir a jornada principal.",
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
                    <p className="text-3xl font-bold text-white">1 tela</p>
                    <p className="mt-1 text-sm text-slate-400">
                      para ler saude, uso e operacao da conta
                    </p>
                  </div>
                  <div>
                    <p className="text-3xl font-bold text-white">3 camadas</p>
                    <p className="mt-1 text-sm text-slate-400">
                      visao geral, produtividade e administracao
                    </p>
                  </div>
                </div>
              </div>
              {[
                [
                  "Estado da conexao",
                  "QR, status, pareamento e retomada da instancia em tempo real.",
                ],
                [
                  "Mensageria operacional",
                  "Envios, inbox e campanhas com leitura clara para equipe comercial e suporte.",
                ],
                [
                  "Governanca",
                  "Equipe, API keys, auditoria e webhooks organizados em modulos de administracao.",
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
      </div>
    </main>
  );
}
