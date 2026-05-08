"use client";

import { LoaderCircle } from "lucide-react";
import type { ReactNode } from "react";

import type { TenantMember, UserRole } from "@/lib/types";

const ROLE_LABELS: Record<UserRole, string> = {
  owner: "Owner",
  admin: "Admin",
  operator: "Operator",
  viewer: "Viewer",
};

export function MetricCard({
  icon,
  label,
  value,
  hint,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  hint: string;
}) {
  return (
    <div className="metric-shell h-full">
      <div className="flex items-start justify-between gap-4">
        <span className="panel-title max-w-[70%] break-words">{label}</span>
        <div className="shrink-0 rounded-2xl border border-glow/20 bg-glow/10 p-2 text-glow">
          {icon}
        </div>
      </div>
      <div className="mt-5 flex min-h-24 flex-col justify-between gap-3">
        <p className="text-2xl font-bold text-white">{value}</p>
        <p className="text-sm leading-6 text-slate-400">{hint}</p>
      </div>
    </div>
  );
}

export function FormPanel({
  title,
  icon,
  action,
  children,
}: {
  title: string;
  icon: ReactNode;
  action: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="panel overflow-hidden p-6">
      <div className="flex flex-col gap-4">
        <div className="min-w-0 flex items-start gap-3">
          <div className="shrink-0 rounded-2xl border border-white/10 bg-white/5 p-2 text-glow">
            {icon}
          </div>
          <div className="min-w-0">
            <p className="panel-title break-words">{title}</p>
          </div>
        </div>
        {action ? (
          <div className="flex w-full justify-start sm:justify-end [&>*]:w-full sm:[&>*]:w-auto sm:[&>*]:max-w-full">
            {action}
          </div>
        ) : null}
      </div>
      <div className="mt-5 space-y-3">{children}</div>
    </div>
  );
}

export function QueueMetric({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
      <p className="text-xs uppercase tracking-[0.22em] text-slate-500">
        {label}
      </p>
      <p className="mt-2 text-2xl font-semibold text-white">{value}</p>
    </div>
  );
}

export function EmptyState({ label }: { label: string }) {
  return (
    <p className="rounded-2xl border border-dashed border-white/10 px-4 py-8 text-center text-sm text-slate-500">
      {label}
    </p>
  );
}

export function MemberRow({
  actorRole,
  currentUserID,
  member,
  onChangeRole,
}: {
  actorRole: UserRole;
  currentUserID: string;
  member: TenantMember;
  onChangeRole: (role: string) => void;
}) {
  const editable =
    (actorRole === "owner" && currentUserID !== member.user_id) ||
    (actorRole === "admin" &&
      currentUserID !== member.user_id &&
      member.role !== "owner" &&
      member.role !== "admin");

  return (
    <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-sm font-semibold text-white">{member.name}</p>
          <p className="mt-1 text-xs text-slate-400">{member.email}</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="badge">{ROLE_LABELS[member.role]}</span>
          <select
            className="input h-10 w-36"
            defaultValue={member.role}
            disabled={!editable}
            onChange={(event) => onChangeRole(event.target.value)}
          >
            <option value="viewer">Viewer</option>
            <option value="operator">Operator</option>
            <option value="admin">Admin</option>
            <option value="owner">Owner</option>
          </select>
        </div>
      </div>
    </div>
  );
}

export function LoadingScreen({
  label,
  action,
}: {
  label: string;
  action?: ReactNode;
}) {
  return (
    <main className="flex min-h-screen items-center justify-center px-6">
      <div className="panel flex w-full max-w-lg flex-col items-center gap-4 p-8 text-center">
        <LoaderCircle className="h-8 w-8 animate-spin text-glow" />
        <p className="text-lg text-white">{label}</p>
        {action}
      </div>
    </main>
  );
}
