"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Cable,
  Copy,
  KeyRound,
  LoaderCircle,
  LogOut,
  MessageSquareShare,
  PlugZap,
  RefreshCcw,
  Send,
  ShieldCheck,
  UsersRound,
  Webhook,
} from "lucide-react";
import { useRouter } from "next/navigation";
import type { ReactNode } from "react";
import {
  useDeferredValue,
  useEffect,
  useMemo,
  useState,
  useTransition,
} from "react";

import {
  api,
  makeMediaDownloadURL,
  makeQRImageURL,
  makeWSURL,
} from "@/lib/api";
import {
  clearAuthToken,
  clearSelectedTenantID,
  getAuthToken,
  getSelectedTenantID,
  setSelectedTenantID,
} from "@/lib/auth";
import type {
  AppEvent,
  CurrentUserResponse,
  Message,
  TenantMember,
  UserRole,
} from "@/lib/types";
import { formatDate } from "@/lib/utils";

const QUERY_KEYS = {
  me: ["me"],
  summary: (tenantID: string) => ["summary", tenantID],
  status: (tenantID: string) => ["status", tenantID],
  messages: (tenantID: string) => ["messages", tenantID],
  webhooks: (tenantID: string) => ["webhooks", tenantID],
  apiKeys: (tenantID: string) => ["apikeys", tenantID],
  usage: (tenantID: string) => ["usage", tenantID],
  members: (tenantID: string) => ["members", tenantID],
};

const ROLE_LABELS: Record<UserRole, string> = {
  owner: "Owner",
  admin: "Admin",
  operator: "Operator",
  viewer: "Viewer",
};

export function DashboardClient() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [token, setToken] = useState("");
  const [selectedTenantID, setTenant] = useState("");
  const [sendForm, setSendForm] = useState({ phone: "", message: "" });
  const [mediaForm, setMediaForm] = useState({
    phone: "",
    type: "image",
    url: "",
    caption: "",
  });
  const [webhookForm, setWebhookForm] = useState({
    url: "",
    events: "message.received,message.sent,message.status,connection.update",
  });
  const [apiKeyLabel, setAPIKeyLabel] = useState("dashboard");
  const [memberForm, setMemberForm] = useState({ email: "", role: "viewer" });
  const [lastSecret, setLastSecret] = useState("");
  const [flash, setFlash] = useState("");
  const [isPending, startTransition] = useTransition();

  useEffect(() => {
    const authToken = getAuthToken();
    if (!authToken) {
      router.replace("/login");
      return;
    }
    setToken(authToken);
    setTenant(getSelectedTenantID());
  }, [router]);

  useEffect(() => {
    function onTokenChange(event: Event) {
      const nextToken = (event as CustomEvent<string>).detail ?? getAuthToken();
      setToken(nextToken);
    }

    window.addEventListener("wsaas:auth-token", onTokenChange as EventListener);
    return () =>
      window.removeEventListener(
        "wsaas:auth-token",
        onTokenChange as EventListener,
      );
  }, []);

  const meQuery = useQuery({
    queryKey: QUERY_KEYS.me,
    queryFn: () => api.me(token, selectedTenantID || undefined),
    enabled: Boolean(token),
  });

  useEffect(() => {
    const me = meQuery.data;
    if (!me) return;
    const activeTenant =
      selectedTenantID || me.tenant?.id || me.memberships?.[0]?.tenant_id || "";
    if (activeTenant && activeTenant !== selectedTenantID) {
      setSelectedTenantID(activeTenant);
      setTenant(activeTenant);
    }
  }, [meQuery.data, selectedTenantID]);

  const tenantID = selectedTenantID || meQuery.data?.tenant?.id || "";
  const tenantHeader = useDeferredValue(tenantID);

  const summaryQuery = useQuery({
    queryKey: QUERY_KEYS.summary(tenantHeader),
    queryFn: () => api.tenantSummary(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const statusQuery = useQuery({
    queryKey: QUERY_KEYS.status(tenantHeader),
    queryFn: () => api.whatsappStatus(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const messagesQuery = useQuery({
    queryKey: QUERY_KEYS.messages(tenantHeader),
    queryFn: () => api.messages(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const webhooksQuery = useQuery({
    queryKey: QUERY_KEYS.webhooks(tenantHeader),
    queryFn: () => api.webhooks(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const apiKeysQuery = useQuery({
    queryKey: QUERY_KEYS.apiKeys(tenantHeader),
    queryFn: () => api.apiKeys(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const usageQuery = useQuery({
    queryKey: QUERY_KEYS.usage(tenantHeader),
    queryFn: () => api.usage(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const membersQuery = useQuery({
    queryKey: QUERY_KEYS.members(tenantHeader),
    queryFn: () => api.members(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });

  useEffect(() => {
    if (!token || !tenantHeader) return;
    const socket = new WebSocket(makeWSURL(token, tenantHeader));
    socket.onmessage = (event) => {
      const payload = JSON.parse(event.data) as AppEvent<Message>;
      if (
        payload.type === "message.received" ||
        payload.type === "message.sent" ||
        payload.type === "message.status"
      ) {
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.messages(tenantHeader),
        });
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.usage(tenantHeader),
        });
      }
      if (payload.type === "connection.update") {
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.status(tenantHeader),
        });
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.summary(tenantHeader),
        });
      }
    };
    return () => socket.close();
  }, [queryClient, tenantHeader, token]);

  const sendMessage = useMutation({
    mutationFn: () => api.sendMessage(token, tenantHeader, sendForm),
    onSuccess: () => {
      setSendForm({ phone: "", message: "" });
      notify("Mensagem enviada.");
      invalidateTenant(tenantHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const sendMedia = useMutation({
    mutationFn: () => api.sendMedia(token, tenantHeader, mediaForm),
    onSuccess: () => {
      setMediaForm({ phone: "", type: "image", url: "", caption: "" });
      notify("Midia enviada.");
      invalidateTenant(tenantHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const createWebhook = useMutation({
    mutationFn: () =>
      api.createWebhook(token, tenantHeader, {
        url: webhookForm.url,
        events: webhookForm.events
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean),
      }),
    onSuccess: () => {
      setWebhookForm((current) => ({ ...current, url: "" }));
      notify("Webhook registrado.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.webhooks(tenantHeader),
      });
    },
    onError: handleMutationError,
  });

  const createAPIKey = useMutation({
    mutationFn: () =>
      api.createAPIKey(token, tenantHeader, { label: apiKeyLabel }),
    onSuccess: (data) => {
      setLastSecret(data.api_key);
      notify("API key criada.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.apiKeys(tenantHeader),
      });
    },
    onError: handleMutationError,
  });

  const addMember = useMutation({
    mutationFn: () => api.addMember(token, tenantHeader, memberForm),
    onSuccess: () => {
      setMemberForm({ email: "", role: "viewer" });
      notify("Membro adicionado ao workspace.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.members(tenantHeader),
      });
    },
    onError: handleMutationError,
  });

  const updateMemberRole = useMutation({
    mutationFn: ({ memberID, role }: { memberID: string; role: string }) =>
      api.updateMemberRole(token, tenantHeader, memberID, { role }),
    onSuccess: () => {
      notify("Role atualizada.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.members(tenantHeader),
      });
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.me });
    },
    onError: handleMutationError,
  });

  const statusAction = useMutation({
    mutationFn: async (action: "connect" | "disconnect" | "logout") => {
      if (action === "connect") return api.whatsappConnect(token, tenantHeader);
      if (action === "disconnect")
        return api.whatsappDisconnect(token, tenantHeader);
      return api.whatsappLogout(token, tenantHeader);
    },
    onSuccess: () => {
      invalidateTenant(tenantHeader, queryClient);
    },
    onError: handleMutationError,
  });

  async function signOut() {
    try {
      await api.logout(token);
    } catch {}
    clearAuthToken();
    clearSelectedTenantID();
    router.replace("/login");
  }

  function notify(message: string) {
    setFlash(message);
    window.setTimeout(() => setFlash(""), 2400);
  }

  function handleMutationError(error: unknown) {
    const message =
      error instanceof Error ? error.message : "Falha inesperada.";
    setFlash(message);
  }

  const currentUser = meQuery.data as CurrentUserResponse | undefined;
  const currentMembership =
    currentUser?.memberships.find(
      (membership) => membership.tenant_id === tenantHeader,
    ) ?? currentUser?.membership;
  const usage = usageQuery.data;
  const usagePercent = useMemo(() => {
    const limit =
      summaryQuery.data?.plan?.plan_id === "plan_pro"
        ? 100000
        : summaryQuery.data?.plan?.plan_id === "plan_growth"
          ? 10000
          : 1000;
    const sent = usage?.sent ?? 0;
    return Math.min(100, Math.round((sent / limit) * 100));
  }, [summaryQuery.data?.plan?.plan_id, usage?.sent]);

  if (!token || meQuery.isLoading) {
    return <LoadingScreen label="Carregando console" />;
  }

  if (meQuery.error) {
    return (
      <LoadingScreen
        label="Sua sessao nao esta valida."
        action={
          <button
            className="button-primary"
            onClick={() => router.replace("/login")}
          >
            Voltar para login
          </button>
        }
      />
    );
  }

  return (
    <main
      className="relative min-h-screen overflow-hidden px-4 py-6 lg:px-8"
      data-testid="dashboard-root"
    >
      <div className="grid-background absolute inset-0 opacity-35" />
      <div className="relative mx-auto flex max-w-7xl flex-col gap-6">
        <header className="panel flex flex-col gap-5 p-6 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="panel-title">Workspace Control Surface</p>
            <h1 className="mt-2 text-3xl font-bold text-white">
              {summaryQuery.data?.tenant?.name ??
                currentUser?.tenant?.name ??
                "Console"}
            </h1>
            <p className="mt-2 text-sm text-slate-400">
              Operando como{" "}
              <span className="text-white">{currentUser?.user.name}</span> •{" "}
              {ROLE_LABELS[currentMembership?.role ?? "viewer"]}
            </p>
          </div>
          <div className="flex flex-col gap-3 lg:items-end">
            <select
              className="input min-w-64"
              value={tenantHeader}
              onChange={(event) =>
                startTransition(() => {
                  setTenant(event.target.value);
                  setSelectedTenantID(event.target.value);
                })
              }
            >
              {currentUser?.memberships.map((membership) => (
                <option key={membership.id} value={membership.tenant_id}>
                  {membership.tenant_id} • {ROLE_LABELS[membership.role]}
                </option>
              ))}
            </select>
            <div className="flex gap-3">
              <button
                className="button-secondary"
                onClick={() => invalidateTenant(tenantHeader, queryClient)}
                type="button"
              >
                <RefreshCcw className="mr-2 h-4 w-4" />
                Atualizar
              </button>
              <button className="button-danger" onClick={signOut} type="button">
                <LogOut className="mr-2 h-4 w-4" />
                Sair
              </button>
            </div>
          </div>
        </header>

        {flash ? (
          <div className="rounded-2xl border border-glow/20 bg-glow/10 px-4 py-3 text-sm text-glow">
            {flash}
          </div>
        ) : null}

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            icon={<Cable className="h-5 w-5" />}
            label="Sessao WhatsApp"
            value={statusQuery.data?.status ?? "unknown"}
            hint={statusQuery.data?.phone || "Sem numero pareado"}
          />
          <MetricCard
            icon={<Activity className="h-5 w-5" />}
            label="Uso do mes"
            value={`${usage?.sent ?? 0} enviadas`}
            hint={`${usage?.received ?? 0} recebidas`}
          />
          <MetricCard
            icon={<ShieldCheck className="h-5 w-5" />}
            label="Plano"
            value={summaryQuery.data?.plan?.plan_id ?? "starter"}
            hint={`vencimento ${formatDate(summaryQuery.data?.plan?.period_end)}`}
          />
          <MetricCard
            icon={<MessageSquareShare className="h-5 w-5" />}
            label="Mensagens"
            value={`${messagesQuery.data?.length ?? 0} no feed`}
            hint={isPending ? "trocando tenant..." : "realtime ativo"}
          />
        </section>

        <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
          <div className="grid gap-6">
            <div className="panel p-6">
              <div className="flex flex-wrap items-center justify-between gap-4">
                <div>
                  <p className="panel-title">Conexao WhatsApp</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Pareamento, status e QR
                  </h2>
                </div>
                <div className="flex flex-wrap gap-3">
                  <button
                    className="button-primary"
                    onClick={() => statusAction.mutate("connect")}
                    type="button"
                  >
                    <PlugZap className="mr-2 h-4 w-4" />
                    Conectar
                  </button>
                  <button
                    className="button-secondary"
                    onClick={() => statusAction.mutate("disconnect")}
                    type="button"
                  >
                    Desconectar
                  </button>
                  <button
                    className="button-danger"
                    onClick={() => statusAction.mutate("logout")}
                    type="button"
                  >
                    Desparear
                  </button>
                </div>
              </div>

              <div className="mt-6 grid gap-5 lg:grid-cols-[1fr_280px]">
                <div className="rounded-3xl border border-white/10 bg-slate-950/60 p-5">
                  <div className="flex items-center justify-between">
                    <span className="badge">
                      {statusQuery.data?.status ?? "disconnected"}
                    </span>
                    {statusAction.isPending ? (
                      <LoaderCircle className="h-5 w-5 animate-spin text-glow" />
                    ) : null}
                  </div>
                  <dl className="mt-4 space-y-3 text-sm text-slate-300">
                    <div className="flex justify-between gap-4">
                      <dt>Numero</dt>
                      <dd>{statusQuery.data?.phone || "-"}</dd>
                    </div>
                    <div className="flex justify-between gap-4">
                      <dt>Atualizado</dt>
                      <dd>{formatDate(statusQuery.data?.updated_at)}</dd>
                    </div>
                    <div className="flex justify-between gap-4">
                      <dt>Erro</dt>
                      <dd className="max-w-52 truncate">
                        {statusQuery.data?.last_error || "-"}
                      </dd>
                    </div>
                  </dl>
                  <div className="mt-6">
                    <div className="mb-2 flex items-center justify-between text-xs uppercase tracking-[0.22em] text-slate-400">
                      <span>Uso do plano</span>
                      <span>{usagePercent}%</span>
                    </div>
                    <div className="h-3 rounded-full bg-white/10">
                      <div
                        className="h-3 rounded-full bg-gradient-to-r from-glow to-neon"
                        style={{ width: `${usagePercent}%` }}
                      />
                    </div>
                  </div>
                </div>

                <div className="rounded-3xl border border-white/10 bg-slate-950/60 p-5">
                  <p className="text-sm font-semibold text-white">
                    QR Code atual
                  </p>
                  <div className="mt-4 flex min-h-64 items-center justify-center rounded-3xl border border-dashed border-white/10 bg-slate-900/80 p-4">
                    {statusQuery.data?.qr_code ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        alt="WhatsApp QR"
                        className="h-56 w-56 rounded-2xl bg-white p-3"
                        src={makeQRImageURL(token, tenantHeader)}
                      />
                    ) : (
                      <p className="text-center text-sm leading-6 text-slate-400">
                        Sem QR pendente. Gere uma nova conexao para parear um
                        dispositivo.
                      </p>
                    )}
                  </div>
                </div>
              </div>
            </div>

            <div className="grid gap-6 lg:grid-cols-2">
              <FormPanel
                title="Enviar texto"
                icon={<Send className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    data-testid="send-submit"
                    disabled={sendMessage.isPending}
                    onClick={() => sendMessage.mutate()}
                    type="button"
                  >
                    Enviar texto
                  </button>
                }
              >
                <input
                  className="input"
                  data-testid="send-phone"
                  placeholder="Telefone com DDI"
                  value={sendForm.phone}
                  onChange={(e) =>
                    setSendForm((current) => ({
                      ...current,
                      phone: e.target.value,
                    }))
                  }
                />
                <textarea
                  className="input min-h-28 py-3"
                  data-testid="send-message"
                  placeholder="Mensagem"
                  value={sendForm.message}
                  onChange={(e) =>
                    setSendForm((current) => ({
                      ...current,
                      message: e.target.value,
                    }))
                  }
                />
              </FormPanel>

              <FormPanel
                title="Enviar midia por URL"
                icon={<MessageSquareShare className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={sendMedia.isPending}
                    onClick={() => sendMedia.mutate()}
                    type="button"
                  >
                    Enviar midia
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="Telefone com DDI"
                  value={mediaForm.phone}
                  onChange={(e) =>
                    setMediaForm((current) => ({
                      ...current,
                      phone: e.target.value,
                    }))
                  }
                />
                <select
                  className="input"
                  value={mediaForm.type}
                  onChange={(e) =>
                    setMediaForm((current) => ({
                      ...current,
                      type: e.target.value,
                    }))
                  }
                >
                  <option value="image">Imagem</option>
                  <option value="video">Video</option>
                  <option value="audio">Audio</option>
                  <option value="document">Documento</option>
                </select>
                <input
                  className="input"
                  placeholder="https://..."
                  value={mediaForm.url}
                  onChange={(e) =>
                    setMediaForm((current) => ({
                      ...current,
                      url: e.target.value,
                    }))
                  }
                />
                <input
                  className="input"
                  placeholder="Legenda opcional"
                  value={mediaForm.caption}
                  onChange={(e) =>
                    setMediaForm((current) => ({
                      ...current,
                      caption: e.target.value,
                    }))
                  }
                />
              </FormPanel>
            </div>

            <div className="panel p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="panel-title">Inbox Operacional</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Mensagens recentes
                  </h2>
                </div>
                {messagesQuery.isFetching ? (
                  <LoaderCircle className="h-4 w-4 animate-spin text-glow" />
                ) : null}
              </div>
              <div className="mt-6 space-y-3">
                {messagesQuery.data?.length ? (
                  messagesQuery.data.slice(0, 12).map((message) => (
                    <div
                      key={message.id}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div className="flex items-center gap-3">
                          <span className="badge">{message.direction}</span>
                          <span className="text-sm font-medium text-white">
                            {message.phone}
                          </span>
                          <span className="text-xs text-slate-500">
                            {message.type}
                          </span>
                        </div>
                        <div className="flex items-center gap-4">
                          <span className="text-xs text-slate-500">
                            {formatDate(message.sent_at)}
                          </span>
                          {message.direct_path ? (
                            <a
                              className="text-xs font-medium text-glow hover:underline"
                              href={makeMediaDownloadURL(
                                token,
                                tenantHeader,
                                message.id,
                              )}
                              target="_blank"
                            >
                              Baixar midia
                            </a>
                          ) : null}
                        </div>
                      </div>
                      <p className="mt-3 text-sm leading-6 text-slate-300">
                        {message.body || "[sem corpo textual]"}
                      </p>
                    </div>
                  ))
                ) : (
                  <EmptyState label="Nenhuma mensagem ainda." />
                )}
              </div>
            </div>
          </div>

          <div className="grid gap-6">
            <FormPanel
              title="Webhooks"
              icon={<Webhook className="h-4 w-4" />}
              action={
                <button
                  className="button-primary"
                  disabled={createWebhook.isPending}
                  onClick={() => createWebhook.mutate()}
                  type="button"
                >
                  Registrar webhook
                </button>
              }
            >
              <input
                className="input"
                placeholder="https://sua-app.com/hook"
                value={webhookForm.url}
                onChange={(e) =>
                  setWebhookForm((current) => ({
                    ...current,
                    url: e.target.value,
                  }))
                }
              />
              <input
                className="input"
                placeholder="message.received,message.sent"
                value={webhookForm.events}
                onChange={(e) =>
                  setWebhookForm((current) => ({
                    ...current,
                    events: e.target.value,
                  }))
                }
              />
              <div className="space-y-3">
                {webhooksQuery.data?.length ? (
                  webhooksQuery.data.map((webhook) => (
                    <div
                      key={webhook.id}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <p className="text-sm font-semibold text-white">
                        {webhook.url}
                      </p>
                      <p className="mt-2 text-xs text-slate-400">
                        {webhook.events.join(" • ")}
                      </p>
                      <button
                        className="mt-4 text-xs font-medium text-danger hover:underline"
                        onClick={() =>
                          api
                            .deleteWebhook(token, tenantHeader, webhook.id)
                            .then(() => {
                              notify("Webhook removido.");
                              queryClient.invalidateQueries({
                                queryKey: QUERY_KEYS.webhooks(tenantHeader),
                              });
                            })
                        }
                        type="button"
                      >
                        Remover
                      </button>
                    </div>
                  ))
                ) : (
                  <EmptyState label="Sem webhooks registrados." />
                )}
              </div>
            </FormPanel>

            <FormPanel
              title="API Keys"
              icon={<KeyRound className="h-4 w-4" />}
              action={
                <button
                  className="button-primary"
                  disabled={createAPIKey.isPending}
                  onClick={() => createAPIKey.mutate()}
                  type="button"
                >
                  Criar API key
                </button>
              }
            >
              <input
                className="input"
                placeholder="Label da credencial"
                value={apiKeyLabel}
                onChange={(e) => setAPIKeyLabel(e.target.value)}
              />
              {lastSecret ? (
                <div className="rounded-2xl border border-ember/30 bg-ember/10 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-xs uppercase tracking-[0.22em] text-ember">
                        Mostrado uma vez
                      </p>
                      <p className="mt-2 break-all text-sm text-white">
                        {lastSecret}
                      </p>
                    </div>
                    <button
                      className="button-secondary"
                      onClick={() => navigator.clipboard.writeText(lastSecret)}
                      type="button"
                    >
                      <Copy className="mr-2 h-4 w-4" />
                      Copiar
                    </button>
                  </div>
                </div>
              ) : null}
              <div className="space-y-3">
                {apiKeysQuery.data?.length ? (
                  apiKeysQuery.data.map((key) => (
                    <div
                      key={key.id}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <div className="flex items-center justify-between gap-4">
                        <div>
                          <p className="text-sm font-semibold text-white">
                            {key.label}
                          </p>
                          <p className="mt-1 text-xs text-slate-400">
                            {key.key_prefix} • ultimo uso{" "}
                            {formatDate(key.last_used)}
                          </p>
                        </div>
                        <button
                          className="text-xs font-medium text-danger hover:underline"
                          onClick={() =>
                            api
                              .deleteAPIKey(token, tenantHeader, key.id)
                              .then(() => {
                                notify("API key revogada.");
                                queryClient.invalidateQueries({
                                  queryKey: QUERY_KEYS.apiKeys(tenantHeader),
                                });
                              })
                          }
                          type="button"
                        >
                          Revogar
                        </button>
                      </div>
                    </div>
                  ))
                ) : (
                  <EmptyState label="Nenhuma credencial disponivel." />
                )}
              </div>
            </FormPanel>

            <FormPanel
              title="Equipe do Workspace"
              icon={<UsersRound className="h-4 w-4" />}
              action={
                <button
                  className="button-primary"
                  disabled={addMember.isPending}
                  onClick={() => addMember.mutate()}
                  type="button"
                >
                  Adicionar membro
                </button>
              }
            >
              <input
                className="input"
                data-testid="member-email"
                placeholder="email do usuario ja cadastrado"
                value={memberForm.email}
                onChange={(e) =>
                  setMemberForm((current) => ({
                    ...current,
                    email: e.target.value,
                  }))
                }
              />
              <select
                className="input"
                value={memberForm.role}
                onChange={(e) =>
                  setMemberForm((current) => ({
                    ...current,
                    role: e.target.value,
                  }))
                }
              >
                <option value="viewer">Viewer</option>
                <option value="operator">Operator</option>
                <option value="admin">Admin</option>
                <option value="owner">Owner</option>
              </select>
              <div className="space-y-3">
                {membersQuery.data?.length ? (
                  membersQuery.data.map((member) => (
                    <MemberRow
                      key={member.id}
                      actorRole={currentMembership?.role ?? "viewer"}
                      currentUserID={currentUser?.user.id ?? ""}
                      member={member}
                      onChangeRole={(role) =>
                        updateMemberRole.mutate({ memberID: member.id, role })
                      }
                    />
                  ))
                ) : (
                  <EmptyState label="Sem membros adicionais neste workspace." />
                )}
              </div>
            </FormPanel>
          </div>
        </section>
      </div>
    </main>
  );
}

function invalidateTenant(
  tenantID: string,
  queryClient: ReturnType<typeof useQueryClient>,
) {
  if (!tenantID) return;
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.summary(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.status(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.messages(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.webhooks(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.apiKeys(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.usage(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members(tenantID) });
}

function MetricCard({
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
    <div className="panel p-5">
      <div className="flex items-center justify-between">
        <span className="panel-title">{label}</span>
        <div className="rounded-2xl border border-glow/20 bg-glow/10 p-2 text-glow">
          {icon}
        </div>
      </div>
      <p className="mt-5 text-2xl font-bold text-white">{value}</p>
      <p className="mt-2 text-sm text-slate-400">{hint}</p>
    </div>
  );
}

function FormPanel({
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
    <div className="panel p-6">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="rounded-2xl border border-white/10 bg-white/5 p-2 text-glow">
            {icon}
          </div>
          <div>
            <p className="panel-title">{title}</p>
          </div>
        </div>
        {action}
      </div>
      <div className="mt-5 space-y-3">{children}</div>
    </div>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <p className="rounded-2xl border border-dashed border-white/10 px-4 py-8 text-center text-sm text-slate-500">
      {label}
    </p>
  );
}

function MemberRow({
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

function LoadingScreen({
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
