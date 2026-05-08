"use client";

import type { ReactNode } from "react";

export interface DashboardExecutiveStat {
  label: string;
  value: string;
  hint: string;
}

export function DashboardHeader({
  title,
  operatorName,
  operatorRole,
  controls,
}: {
  title: string;
  operatorName?: string;
  operatorRole?: string;
  controls?: ReactNode;
}) {
  return (
    <div className="topbar">
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-semibold text-white">{title}</p>
        {operatorName ? (
          <p className="truncate text-xs text-slate-500">
            {operatorName} · {operatorRole}
          </p>
        ) : null}
      </div>
      {controls}
    </div>
  );
}
