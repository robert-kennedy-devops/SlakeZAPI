"use client";

import type { ReactNode } from "react";

export interface DashboardNavLink {
  id: string;
  label: string;
  icon: ReactNode;
}

export function OverviewModule({
  links,
  metrics,
}: {
  links: DashboardNavLink[];
  metrics: ReactNode;
}) {
  return (
    <>
      <section className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        <div className="metric-shell">
          <p className="section-kicker">Leitura executiva</p>
          <h2 className="mt-3 text-2xl font-semibold text-white">
            O que o cliente precisa enxergar primeiro
          </h2>
          <p className="mt-3 max-w-2xl text-sm leading-7 text-slate-400">
            Antes de explorar formularios e configuracoes, o dashboard destaca
            status da conexao, atividade do workspace, pendencias de atendimento
            e integracoes ativas.
          </p>
        </div>
        <div className="metric-shell">
          <p className="section-kicker">Mapa do painel</p>
          <div className="mt-4 flex flex-wrap gap-3">
            {links.map((section) => (
              <a key={section.id} className="nav-chip" href={`#${section.id}`}>
                {section.icon}
                {section.label}
              </a>
            ))}
          </div>
        </div>
      </section>

      <section
        className="grid gap-4 md:grid-cols-2 xl:grid-cols-4"
        id="overview"
      >
        {metrics}
      </section>
    </>
  );
}
