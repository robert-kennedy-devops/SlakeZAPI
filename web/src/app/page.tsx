import Link from "next/link";

export default function HomePage() {
  return (
    <main className="relative overflow-hidden">
      <div className="grid-background absolute inset-0 opacity-40" />
      <div className="relative mx-auto flex min-h-screen max-w-7xl flex-col justify-between px-6 py-10 lg:px-10">
        <header className="flex items-center justify-between">
          <div>
            <p className="panel-title">SlakeZAPI</p>
            <h1 className="mt-2 text-2xl font-bold text-white">Console de Operação WhatsApp</h1>
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

        <section className="grid gap-10 py-16 lg:grid-cols-[1.1fr_0.9fr] lg:items-center">
          <div>
            <p className="badge border-glow/20 bg-glow/10 text-glow">Realtime • Multi-tenant • WhatsApp</p>
            <h2 className="mt-6 max-w-3xl text-5xl font-bold leading-tight text-white">
              Um painel tech, moderno e pronto para operar conexoes, mensagens, webhooks e uso em um so lugar.
            </h2>
            <p className="mt-6 max-w-2xl text-lg leading-8 text-slate-300">
              Interface para equipes operarem sua infraestrutura WhatsApp com login de usuario, permissao por papel,
              QR em tempo real, inbox, automacoes e observabilidade.
            </p>
            <div className="mt-8 flex flex-wrap gap-4">
              <Link className="button-primary" href="/signup">
                Comecar agora
              </Link>
              <Link className="button-secondary" href="/login">
                Acessar dashboard
              </Link>
            </div>
          </div>

          <div className="panel relative p-6">
            <div className="absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-glow/60 to-transparent" />
            <div className="grid gap-4">
              {[
                ["Estado da conexao", "QR, status e pareamento monitorados em tempo real."],
                ["Mensageria operacional", "Envio de texto e midia com fila visual e historico por tenant."],
                ["Webhooks e credenciais", "Controle fino de eventos, API keys e seguranca por workspace."],
              ].map(([title, description]) => (
                <div key={title} className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
                  <p className="text-base font-semibold text-white">{title}</p>
                  <p className="mt-2 text-sm leading-6 text-slate-400">{description}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
