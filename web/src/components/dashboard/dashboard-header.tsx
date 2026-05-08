"use client";

import type { ReactNode } from "react";

export interface DashboardExecutiveStat {
  label: string;
  value: string;
  hint: string;
}

export function DashboardHeader({
  title,
  description,
  operatorName,
  operatorRole,
  badges,
  executiveStats,
  controls,
}: {
  title: string;
  description: string;
  operatorName?: string;
  operatorRole?: string;
  badges: string[];
  executiveStats: DashboardExecutiveStat[];
  controls: ReactNode;
}) {
  return (
    <header className="panel p-6">
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(340px,0.9fr)]">
        <div className="min-w-0">
          <p className="section-kicker">Workspace Command Center</p>
          <h1 className="mt-2 text-3xl font-bold text-white">{title}</h1>
          <p className="mt-4 max-w-2xl text-base leading-7 text-slate-300">
            {description}
          </p>
          <p className="mt-4 text-sm text-slate-400">
            Operando como <span className="text-white">{operatorName}</span> •{" "}
            {operatorRole}
          </p>
          <div className="mt-5 flex flex-wrap gap-3">
            {badges.map((badge) => (
              <span key={badge} className="badge">
                {badge}
              </span>
            ))}
          </div>
          <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {executiveStats.map((item) => (
              <div key={item.label} className="surface-muted p-4">
                <p className="text-xs uppercase tracking-[0.22em] text-slate-500">
                  {item.label}
                </p>
                <p className="mt-3 text-2xl font-bold text-white">
                  {item.value}
                </p>
                <p className="mt-2 text-sm text-slate-400">{item.hint}</p>
              </div>
            ))}
          </div>
        </div>
        {controls}
      </div>
    </header>
  );
}
