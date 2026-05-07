"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Cable,
  CirclePlay,
  Copy,
  FileCode2,
  Files,
  LayoutList,
  KeyRound,
  LoaderCircle,
  LogOut,
  MessageSquareShare,
  MessagesSquare,
  RadioTower,
  PlugZap,
  RefreshCcw,
  Send,
  ShieldCheck,
  Smartphone,
  SquareMousePointer,
  UsersRound,
  Webhook,
} from "lucide-react";
import { useRouter } from "next/navigation";
import type { ChangeEvent, ReactNode } from "react";
import {
  useDeferredValue,
  useEffect,
  useId,
  useMemo,
  useState,
  useTransition,
} from "react";

import {
  api,
  makeAPIDocURL,
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
  ResolvedContact,
  CurrentUserResponse,
  Message,
  TenantMember,
  UserRole,
} from "@/lib/types";
import { formatDate } from "@/lib/utils";

const QUERY_KEYS = {
  me: ["me"],
  summary: (tenantID: string) => ["summary", tenantID],
  instances: (tenantID: string) => ["instances", tenantID],
  campaigns: (tenantID: string, instanceID: string) => [
    "campaigns",
    tenantID,
    instanceID,
  ],
  status: (tenantID: string, instanceID: string) => ["status", tenantID, instanceID],
  messages: (tenantID: string, instanceID: string) => ["messages", tenantID, instanceID],
  conversations: (tenantID: string, instanceID: string) => [
    "conversations",
    tenantID,
    instanceID,
  ],
  queue: (tenantID: string) => ["queue", tenantID],
  queueDeadLetters: (tenantID: string) => ["queue-dead-letters", tenantID],
  groups: (tenantID: string, instanceID: string) => ["groups", tenantID, instanceID],
  webhooks: (tenantID: string, instanceID: string) => ["webhooks", tenantID, instanceID],
  webhookDeliveries: (tenantID: string, instanceID: string) => [
    "webhook-deliveries",
    tenantID,
    instanceID,
  ],
  apiKeys: (tenantID: string) => ["apikeys", tenantID],
  usage: (tenantID: string) => ["usage", tenantID],
  members: (tenantID: string) => ["members", tenantID],
  audit: (tenantID: string, instanceID: string) => ["audit", tenantID, instanceID],
};

const ROLE_LABELS: Record<UserRole, string> = {
  owner: "Owner",
  admin: "Admin",
  operator: "Operator",
  viewer: "Viewer",
};

const PHONE_PATTERN = /(?:\+?\d[\d\s().-]{7,}\d)/g;

export function DashboardClient() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const csvInputID = useId();
  const [token, setToken] = useState("");
  const [selectedTenantID, setTenant] = useState("");
  const [selectedInstanceID, setSelectedInstanceID] = useState("");
  const [sendForm, setSendForm] = useState({ phone: "", message: "" });
  const [contactInput, setContactInput] = useState("");
  const [bulkMessage, setBulkMessage] = useState("");
  const [resolvedContacts, setResolvedContacts] = useState<ResolvedContact[]>(
    [],
  );
  const [selectedBulkPhones, setSelectedBulkPhones] = useState<string[]>([]);
  const [mediaForm, setMediaForm] = useState({
    phone: "",
    type: "image",
    url: "",
    caption: "",
  });
  const [interactiveForm, setInteractiveForm] = useState({
    phone: "",
    type: "buttons",
    header: "",
    body: "",
    footer: "",
    buttonText: "Ver opcoes",
    buttonsText: "sales:Comercial\nsupport:Suporte",
    listSectionsText:
      "Atendimento|billing:Financeiro:2a via,tech:Suporte tecnico:Abertura de chamado",
    pollOptionsText: "Sim\nNao",
    maxSelect: 1,
  });
  const [groupForm, setGroupForm] = useState({ groupJID: "", message: "" });
  const [statusForm, setStatusForm] = useState({
    type: "text",
    message: "",
    url: "",
    caption: "",
  });
  const [webhookForm, setWebhookForm] = useState({
    url: "",
    events: "message.received,message.sent,message.status,connection.update",
  });
  const [conversationDrafts, setConversationDrafts] = useState<
    Record<string, { state: "open" | "pending" | "resolved"; note: string }>
  >({});
  const [apiKeyLabel, setAPIKeyLabel] = useState("dashboard");
  const [instanceName, setInstanceName] = useState("");
  const [campaignForm, setCampaignForm] = useState({
    name: "",
    message: "",
    scheduledAt: "",
  });
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
  const instancesQuery = useQuery({
    queryKey: QUERY_KEYS.instances(tenantHeader),
    queryFn: () => api.instances(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const instanceID =
    selectedInstanceID ||
    instancesQuery.data?.[0]?.id ||
    summaryQuery.data?.instances?.[0]?.id ||
    "";
  const instanceHeader = useDeferredValue(instanceID);
  const statusQuery = useQuery({
    queryKey: QUERY_KEYS.status(tenantHeader, instanceHeader),
    queryFn: () => api.whatsappStatus(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
  });
  const messagesQuery = useQuery({
    queryKey: QUERY_KEYS.messages(tenantHeader, instanceHeader),
    queryFn: () => api.messages(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
  });
  const conversationsQuery = useQuery({
    queryKey: QUERY_KEYS.conversations(tenantHeader, instanceHeader),
    queryFn: () => api.conversations(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
  });
  const groupsQuery = useQuery({
    queryKey: QUERY_KEYS.groups(tenantHeader, instanceHeader),
    queryFn: () => api.groups(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
  });
  const webhooksQuery = useQuery({
    queryKey: QUERY_KEYS.webhooks(tenantHeader, instanceHeader),
    queryFn: () => api.webhooks(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
  });
  const webhookDeliveriesQuery = useQuery({
    queryKey: QUERY_KEYS.webhookDeliveries(tenantHeader, instanceHeader),
    queryFn: () => api.webhookDeliveries(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
    refetchInterval: 5000,
  });
  const campaignsQuery = useQuery({
    queryKey: QUERY_KEYS.campaigns(tenantHeader, instanceHeader),
    queryFn: () => api.campaigns(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader && instanceHeader),
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
  const queueQuery = useQuery({
    queryKey: QUERY_KEYS.queue(tenantHeader),
    queryFn: () => api.queue(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
    refetchInterval: 5000,
  });
  const queueDeadLettersQuery = useQuery({
    queryKey: QUERY_KEYS.queueDeadLetters(tenantHeader),
    queryFn: () => api.queueDeadLetters(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
    refetchInterval: 5000,
  });
  const membersQuery = useQuery({
    queryKey: QUERY_KEYS.members(tenantHeader),
    queryFn: () => api.members(token, tenantHeader),
    enabled: Boolean(token && tenantHeader),
  });
  const auditQuery = useQuery({
    queryKey: QUERY_KEYS.audit(tenantHeader, instanceHeader),
    queryFn: () => api.auditLogs(token, tenantHeader, instanceHeader),
    enabled: Boolean(token && tenantHeader),
    refetchInterval: 10000,
  });

  useEffect(() => {
    if (!token || !tenantHeader || !instanceHeader) return;
    const socket = new WebSocket(makeWSURL(token, tenantHeader, instanceHeader));
    socket.onmessage = (event) => {
      const payload = JSON.parse(event.data) as AppEvent<Message>;
      if (
        payload.type === "message.received" ||
        payload.type === "message.sent" ||
        payload.type === "message.status"
      ) {
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.messages(tenantHeader, instanceHeader),
        });
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.conversations(tenantHeader, instanceHeader),
        });
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.usage(tenantHeader),
        });
      }
      if (payload.type === "connection.update") {
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.status(tenantHeader, instanceHeader),
        });
        queryClient.invalidateQueries({
          queryKey: QUERY_KEYS.summary(tenantHeader),
        });
      }
    };
    return () => socket.close();
  }, [instanceHeader, queryClient, tenantHeader, token]);

  useEffect(() => {
    if (!instanceHeader && instancesQuery.data?.length) {
      setSelectedInstanceID(instancesQuery.data[0].id);
    }
  }, [instanceHeader, instancesQuery.data]);

  const sendMessage = useMutation({
    mutationFn: () => api.sendMessage(token, tenantHeader, instanceHeader, sendForm),
    onSuccess: () => {
      setSendForm({ phone: "", message: "" });
      notify("Mensagem enviada.");
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const sendMedia = useMutation({
    mutationFn: () => api.sendMedia(token, tenantHeader, instanceHeader, mediaForm),
    onSuccess: () => {
      setMediaForm({ phone: "", type: "image", url: "", caption: "" });
      notify("Midia enviada.");
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const sendInteractive = useMutation({
    mutationFn: () =>
      api.sendInteractiveMessage(
        token,
        tenantHeader,
        instanceHeader,
        buildInteractivePayload(interactiveForm),
      ),
    onSuccess: () => {
      notify("Mensagem interativa enviada.");
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const sendGroup = useMutation({
    mutationFn: () =>
      api.sendGroupMessage(token, tenantHeader, instanceHeader, {
        group_jid: groupForm.groupJID,
        message: groupForm.message,
      }),
    onSuccess: () => {
      setGroupForm({ groupJID: "", message: "" });
      notify("Mensagem de grupo enviada.");
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const postStatus = useMutation({
    mutationFn: () =>
      api.postStatus(token, tenantHeader, instanceHeader, {
        type: statusForm.type,
        message: statusForm.message,
        url: statusForm.url || undefined,
        caption: statusForm.caption || undefined,
      }),
    onSuccess: () => {
      setStatusForm({ type: "text", message: "", url: "", caption: "" });
      notify("Status publicado.");
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const resolveContacts = useMutation({
    mutationFn: async () =>
      api.resolveContacts(token, tenantHeader, instanceHeader, {
        phones: parsedContactPhones,
      }),
    onSuccess: (contacts) => {
      setResolvedContacts(contacts);
      setSelectedBulkPhones(
        contacts
          .filter((item) => item.is_whatsapp && item.phone)
          .map((item) => item.phone),
      );
      notify(
        `${contacts.filter((item) => item.is_whatsapp).length} contatos reconhecidos no WhatsApp.`,
      );
    },
    onError: handleMutationError,
  });

  const createInstance = useMutation({
    mutationFn: () => api.createInstance(token, tenantHeader, { name: instanceName }),
    onSuccess: (instance) => {
      setInstanceName("");
      setSelectedInstanceID(instance.id);
      notify("Instancia criada.");
      queryClient.invalidateQueries({ queryKey: QUERY_KEYS.instances(tenantHeader) });
    },
    onError: handleMutationError,
  });

  const createCampaign = useMutation({
    mutationFn: () =>
      api.createCampaign(token, tenantHeader, instanceHeader, {
        name: campaignForm.name,
        message: campaignForm.message,
        scheduled_at: campaignForm.scheduledAt
          ? new Date(campaignForm.scheduledAt).toISOString()
          : undefined,
        recipients: parsedContactPhones.map((phone) => ({ phone })),
      }),
    onSuccess: () => {
      setCampaignForm({ name: "", message: "", scheduledAt: "" });
      notify("Campanha criada.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.campaigns(tenantHeader, instanceHeader),
      });
    },
    onError: handleMutationError,
  });

  const sendBulkMessage = useMutation({
    mutationFn: () =>
      api.sendBulkMessage(token, tenantHeader, instanceHeader, {
        phones: selectedBulkPhones,
        message: bulkMessage,
      }),
    onSuccess: (result) => {
      notify(
        `Disparo concluido: ${result.sent} enviados, ${result.failed} falharam.`,
      );
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const updateConversation = useMutation({
    mutationFn: ({
      phone,
      state,
      note,
    }: {
      phone: string;
      state: "open" | "pending" | "resolved";
      note: string;
    }) =>
      api.updateConversation(token, tenantHeader, instanceHeader, phone, {
        state,
        note,
      }),
    onSuccess: () => {
      notify("Conversa atualizada.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.conversations(tenantHeader, instanceHeader),
      });
    },
    onError: handleMutationError,
  });

  const runCampaign = useMutation({
    mutationFn: (campaignID: string) =>
      api.runCampaign(token, tenantHeader, instanceHeader, campaignID),
    onSuccess: () => {
      notify("Campanha iniciada.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.campaigns(tenantHeader, instanceHeader),
      });
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const createWebhook = useMutation({
    mutationFn: () =>
      api.createWebhook(token, tenantHeader, instanceHeader, {
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
        queryKey: QUERY_KEYS.webhooks(tenantHeader, instanceHeader),
      });
    },
    onError: handleMutationError,
  });

  const replayWebhookDelivery = useMutation({
    mutationFn: (deliveryID: string) =>
      api.replayWebhookDelivery(token, tenantHeader, instanceHeader, deliveryID),
    onSuccess: () => {
      notify("Replay de webhook enfileirado.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.webhookDeliveries(tenantHeader, instanceHeader),
      });
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.queue(tenantHeader),
      });
    },
    onError: handleMutationError,
  });

  const requeueDeadLetter = useMutation({
    mutationFn: (jobID: string) => api.requeueDeadLetter(token, tenantHeader, jobID),
    onSuccess: () => {
      notify("Job recolocado na fila.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.queue(tenantHeader),
      });
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.queueDeadLetters(tenantHeader),
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
      if (action === "connect")
        return api.whatsappConnect(token, tenantHeader, instanceHeader);
      if (action === "disconnect")
        return api.whatsappDisconnect(token, tenantHeader, instanceHeader);
      return api.whatsappLogout(token, tenantHeader, instanceHeader);
    },
    onSuccess: () => {
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
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
    const message = extractErrorMessage(error);
    setFlash(message);
  }

  function toggleBulkPhone(phone: string) {
    setSelectedBulkPhones((current) =>
      current.includes(phone)
        ? current.filter((item) => item !== phone)
        : [...current, phone],
    );
  }

  function selectAllResolved() {
    setSelectedBulkPhones(
      resolvedContacts
        .filter((item) => item.is_whatsapp && item.phone)
        .map((item) => item.phone),
    );
  }

  async function importCSV(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    try {
      const text = await file.text();
      const phones = parsePhonesFromCSV(text);
      if (!phones.length) {
        notify("Nenhum telefone valido encontrado no CSV.");
        return;
      }
      setContactInput(phones.join("\n"));
      notify(`${phones.length} numeros importados do CSV.`);
    } catch (error) {
      handleMutationError(error);
    }
  }

  useEffect(() => {
    if (!conversationsQuery.data?.length) return;
    setConversationDrafts((current) => {
      const next = { ...current };
      for (const conversation of conversationsQuery.data) {
        if (!next[conversation.phone]) {
          next[conversation.phone] = {
            state: conversation.state,
            note: conversation.note ?? "",
          };
        }
      }
      return next;
    });
  }, [conversationsQuery.data]);

  const currentUser = meQuery.data as CurrentUserResponse | undefined;
  const currentMembership =
    currentUser?.memberships.find(
      (membership) => membership.tenant_id === tenantHeader,
    ) ?? currentUser?.membership;
  const usage = usageQuery.data;
  const parsedContactPhones = useMemo(
    () => extractPhonesFromText(contactInput),
    [contactInput],
  );
  const groupsErrorMessage = groupsQuery.error
    ? extractErrorMessage(groupsQuery.error)
    : "";
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
            <select
              className="input min-w-64"
              value={instanceHeader}
              onChange={(event) => setSelectedInstanceID(event.target.value)}
            >
              {instancesQuery.data?.map((instance) => (
                <option key={instance.id} value={instance.id}>
                  {instance.name} • {instance.phone || instance.status}
                </option>
              ))}
            </select>
            <div className="flex gap-3">
              <input
                className="input min-w-64"
                placeholder="Nome da nova instancia"
                value={instanceName}
                onChange={(event) => setInstanceName(event.target.value)}
              />
              <button
                className="button-secondary"
                disabled={createInstance.isPending || !instanceName.trim()}
                onClick={() => createInstance.mutate()}
                type="button"
              >
                <Smartphone className="mr-2 h-4 w-4" />
                Criar instancia
              </button>
            </div>
            <div className="flex gap-3">
              <button
                className="button-secondary"
                onClick={() =>
                  invalidateTenant(tenantHeader, instanceHeader, queryClient)
                }
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
                        src={makeQRImageURL(token, tenantHeader, instanceHeader)}
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

            <div className="grid gap-6 xl:grid-cols-3">
              <FormPanel
                title="Mensagens Interativas"
                icon={<SquareMousePointer className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={sendInteractive.isPending || !interactiveForm.body.trim()}
                    onClick={() => sendInteractive.mutate()}
                    type="button"
                  >
                    Enviar interacao
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="Telefone com DDI"
                  value={interactiveForm.phone}
                  onChange={(event) =>
                    setInteractiveForm((current) => ({
                      ...current,
                      phone: event.target.value,
                    }))
                  }
                />
                <select
                  className="input"
                  value={interactiveForm.type}
                  onChange={(event) =>
                    setInteractiveForm((current) => ({
                      ...current,
                      type: event.target.value,
                    }))
                  }
                >
                  <option value="buttons">Botoes</option>
                  <option value="list">Lista</option>
                  <option value="poll">Enquete</option>
                </select>
                <input
                  className="input"
                  placeholder="Titulo opcional"
                  value={interactiveForm.header}
                  onChange={(event) =>
                    setInteractiveForm((current) => ({
                      ...current,
                      header: event.target.value,
                    }))
                  }
                />
                <textarea
                  className="input min-h-24 py-3"
                  placeholder="Corpo principal"
                  value={interactiveForm.body}
                  onChange={(event) =>
                    setInteractiveForm((current) => ({
                      ...current,
                      body: event.target.value,
                    }))
                  }
                />
                <input
                  className="input"
                  placeholder="Rodape opcional"
                  value={interactiveForm.footer}
                  onChange={(event) =>
                    setInteractiveForm((current) => ({
                      ...current,
                      footer: event.target.value,
                    }))
                  }
                />
                {interactiveForm.type === "buttons" ? (
                  <textarea
                    className="input min-h-28 py-3"
                    placeholder={"id:Titulo\nsupport:Suporte"}
                    value={interactiveForm.buttonsText}
                    onChange={(event) =>
                      setInteractiveForm((current) => ({
                        ...current,
                        buttonsText: event.target.value,
                      }))
                    }
                  />
                ) : null}
                {interactiveForm.type === "list" ? (
                  <>
                    <input
                      className="input"
                      placeholder="Texto do botao"
                      value={interactiveForm.buttonText}
                      onChange={(event) =>
                        setInteractiveForm((current) => ({
                          ...current,
                          buttonText: event.target.value,
                        }))
                      }
                    />
                    <textarea
                      className="input min-h-28 py-3"
                      placeholder="Secao|id:titulo:descricao,id2:titulo2:descricao2"
                      value={interactiveForm.listSectionsText}
                      onChange={(event) =>
                        setInteractiveForm((current) => ({
                          ...current,
                          listSectionsText: event.target.value,
                        }))
                      }
                    />
                  </>
                ) : null}
                {interactiveForm.type === "poll" ? (
                  <>
                    <textarea
                      className="input min-h-24 py-3"
                      placeholder={"Opcao 1\nOpcao 2"}
                      value={interactiveForm.pollOptionsText}
                      onChange={(event) =>
                        setInteractiveForm((current) => ({
                          ...current,
                          pollOptionsText: event.target.value,
                        }))
                      }
                    />
                    <input
                      className="input"
                      min={1}
                      type="number"
                      value={interactiveForm.maxSelect}
                      onChange={(event) =>
                        setInteractiveForm((current) => ({
                          ...current,
                          maxSelect: Number(event.target.value || 1),
                        }))
                      }
                    />
                  </>
                ) : null}
              </FormPanel>

              <FormPanel
                title="Status"
                icon={<RadioTower className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={postStatus.isPending}
                    onClick={() => postStatus.mutate()}
                    type="button"
                  >
                    Publicar
                  </button>
                }
              >
                <select
                  className="input"
                  value={statusForm.type}
                  onChange={(event) =>
                    setStatusForm((current) => ({
                      ...current,
                      type: event.target.value,
                    }))
                  }
                >
                  <option value="text">Texto</option>
                  <option value="image">Imagem</option>
                  <option value="video">Video</option>
                  <option value="audio">Audio</option>
                  <option value="document">Documento</option>
                </select>
                <textarea
                  className="input min-h-24 py-3"
                  placeholder="Mensagem do status"
                  value={statusForm.message}
                  onChange={(event) =>
                    setStatusForm((current) => ({
                      ...current,
                      message: event.target.value,
                    }))
                  }
                />
                {statusForm.type !== "text" ? (
                  <>
                    <input
                      className="input"
                      placeholder="https://arquivo.exemplo"
                      value={statusForm.url}
                      onChange={(event) =>
                        setStatusForm((current) => ({
                          ...current,
                          url: event.target.value,
                        }))
                      }
                    />
                    <input
                      className="input"
                      placeholder="Legenda opcional"
                      value={statusForm.caption}
                      onChange={(event) =>
                        setStatusForm((current) => ({
                          ...current,
                          caption: event.target.value,
                        }))
                      }
                    />
                  </>
                ) : null}
              </FormPanel>

              <FormPanel
                title="Grupos"
                icon={<MessagesSquare className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={sendGroup.isPending || !groupForm.groupJID || !groupForm.message.trim()}
                    onClick={() => sendGroup.mutate()}
                    type="button"
                  >
                    Enviar ao grupo
                  </button>
                }
              >
                <select
                  className="input"
                  value={groupForm.groupJID}
                  onChange={(event) =>
                    setGroupForm((current) => ({
                      ...current,
                      groupJID: event.target.value,
                    }))
                  }
                >
                  <option value="">Selecione um grupo</option>
                  {groupsQuery.data?.map((group) => (
                    <option key={group.jid} value={group.jid}>
                      {group.name} • {group.participant_count} membros
                    </option>
                  ))}
                </select>
                <textarea
                  className="input min-h-24 py-3"
                  placeholder="Mensagem para o grupo"
                  value={groupForm.message}
                  onChange={(event) =>
                    setGroupForm((current) => ({
                      ...current,
                      message: event.target.value,
                    }))
                  }
                />
                <div className="space-y-2">
                  {groupsQuery.data?.slice(0, 5).map((group) => (
                    <button
                      key={group.jid}
                      className="flex w-full items-center justify-between rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-3 text-left text-sm text-slate-300"
                      onClick={() =>
                        setGroupForm((current) => ({
                          ...current,
                          groupJID: group.jid,
                        }))
                      }
                      type="button"
                    >
                      <span>{group.name}</span>
                      <span className="text-xs text-slate-500">
                        {group.participant_count} membros
                      </span>
                    </button>
                  ))}
                  {groupsQuery.isLoading ? (
                    <EmptyState label="Carregando grupos da instancia..." />
                  ) : null}
                  {!groupsQuery.isLoading && groupsErrorMessage ? (
                    <EmptyState label={`Grupos indisponiveis: ${groupsErrorMessage}`} />
                  ) : null}
                  {!groupsQuery.isLoading &&
                  !groupsErrorMessage &&
                  !groupsQuery.data?.length ? (
                    <EmptyState label="Nenhum grupo encontrado para esta instancia." />
                  ) : null}
                </div>
              </FormPanel>
            </div>

            <div className="panel p-6">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <p className="panel-title">Reconhecimento e Disparo</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Selecionar contatos e enviar em massa
                  </h2>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                    Cole uma lista de numeros com DDI. O backend valida quais
                    existem no WhatsApp e voce pode selecionar varios para
                    disparo em lote.
                  </p>
                </div>
                <div className="flex flex-wrap gap-3">
                  <label
                    className="button-secondary cursor-pointer"
                    htmlFor={csvInputID}
                  >
                    Importar CSV
                  </label>
                  <button
                    className="button-secondary"
                    disabled={
                      resolveContacts.isPending || parsedContactPhones.length === 0
                    }
                    onClick={() => {
                      if (!parsedContactPhones.length) {
                        notify("Adicione pelo menos um telefone valido para reconhecer.");
                        return;
                      }
                      resolveContacts.mutate();
                    }}
                    type="button"
                  >
                    Reconhecer contatos
                  </button>
                  <button
                    className="button-primary"
                    disabled={
                      sendBulkMessage.isPending ||
                      selectedBulkPhones.length === 0 ||
                      !bulkMessage.trim()
                    }
                    onClick={() => sendBulkMessage.mutate()}
                    type="button"
                  >
                    Enviar em massa
                  </button>
                </div>
              </div>

              <div className="mt-6 grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
                <div className="space-y-3">
                  <input
                    accept=".csv,text/csv"
                    className="hidden"
                    id={csvInputID}
                    onChange={importCSV}
                    type="file"
                  />
                  <textarea
                    className="input min-h-56 py-3"
                    placeholder={
                      "5511999999999\n5511888888888\n+55 (11) 97777-7777"
                    }
                    value={contactInput}
                    onChange={(event) => setContactInput(event.target.value)}
                  />
                  <textarea
                    className="input min-h-36 py-3"
                    placeholder="Mensagem que sera enviada para os contatos selecionados"
                    value={bulkMessage}
                    onChange={(event) => setBulkMessage(event.target.value)}
                  />
                  <div className="flex flex-wrap gap-3 text-xs text-slate-400">
                    <span>{parsedContactPhones.length} numeros detectados</span>
                    <span>{resolvedContacts.length} contatos analisados</span>
                    <span>{selectedBulkPhones.length} selecionados</span>
                  </div>
                </div>

                <div className="rounded-3xl border border-white/10 bg-slate-950/50 p-5">
                  <div className="mb-4 flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-semibold text-white">
                        Contatos reconhecidos
                      </p>
                      <p className="mt-1 text-xs text-slate-400">
                        Contatos com WhatsApp podem ser selecionados para envio
                        em massa.
                      </p>
                    </div>
                    <button
                      className="button-secondary"
                      disabled={!resolvedContacts.length}
                      onClick={selectAllResolved}
                      type="button"
                    >
                      Selecionar validos
                    </button>
                  </div>

                  <div className="space-y-3">
                    {resolvedContacts.length ? (
                      resolvedContacts.map((contact) => {
                        const checked = selectedBulkPhones.includes(
                          contact.phone,
                        );
                        const disabled = !contact.is_whatsapp || !contact.phone;
                        return (
                          <label
                            key={`${contact.input_phone}-${contact.phone}`}
                            className="flex cursor-pointer items-start gap-3 rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                          >
                            <input
                              checked={checked}
                              className="mt-1"
                              disabled={disabled}
                              onChange={() => toggleBulkPhone(contact.phone)}
                              type="checkbox"
                            />
                            <div className="min-w-0 flex-1">
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="text-sm font-semibold text-white">
                                  {contact.verified_name ||
                                    contact.phone ||
                                    contact.input_phone}
                                </span>
                                <span
                                  className={`badge ${contact.is_whatsapp ? "text-glow" : "text-danger"}`}
                                >
                                  {contact.is_whatsapp
                                    ? "whatsapp"
                                    : "invalido"}
                                </span>
                              </div>
                              <p className="mt-1 text-xs text-slate-400">
                                entrada: {contact.input_phone}
                              </p>
                              <p className="mt-1 text-xs text-slate-500">
                                canonico: {contact.phone || "-"}
                              </p>
                              {contact.error ? (
                                <p className="mt-2 text-xs text-danger">
                                  {contact.error}
                                </p>
                              ) : null}
                            </div>
                          </label>
                        );
                      })
                    ) : (
                      <EmptyState label="Cole os numeros e clique em reconhecer contatos." />
                    )}
                  </div>
                </div>
              </div>
            </div>

            <div className="panel p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="panel-title">Inbox Operacional</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Conversas e mensagens recentes
                  </h2>
                </div>
                {messagesQuery.isFetching ? (
                  <LoaderCircle className="h-4 w-4 animate-spin text-glow" />
                ) : null}
              </div>
              <div className="mt-6 grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
                <div className="space-y-3">
                  {conversationsQuery.data?.length ? (
                    conversationsQuery.data.slice(0, 8).map((conversation) => {
                      const draft = conversationDrafts[conversation.phone] ?? {
                        state: conversation.state,
                        note: conversation.note ?? "",
                      };
                      return (
                        <div
                          key={conversation.id}
                          className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <p className="text-sm font-semibold text-white">
                                {conversation.phone}
                              </p>
                              <p className="mt-1 text-xs text-slate-400">
                                {conversation.last_direction} •{" "}
                                {formatDate(conversation.last_at)}
                              </p>
                            </div>
                            <span className="badge">
                              {conversation.unread_count} nao lidas
                            </span>
                          </div>
                          <p className="mt-3 text-sm text-slate-300">
                            {conversation.last_message_body || "Sem resumo textual"}
                          </p>
                          <div className="mt-4 grid gap-3">
                            <select
                              className="input"
                              value={draft.state}
                              onChange={(event) =>
                                setConversationDrafts((current) => ({
                                  ...current,
                                  [conversation.phone]: {
                                    state: event.target.value as
                                      | "open"
                                      | "pending"
                                      | "resolved",
                                    note: draft.note,
                                  },
                                }))
                              }
                            >
                              <option value="open">Open</option>
                              <option value="pending">Pending</option>
                              <option value="resolved">Resolved</option>
                            </select>
                            <textarea
                              className="input min-h-20 py-3"
                              placeholder="Nota interna"
                              value={draft.note}
                              onChange={(event) =>
                                setConversationDrafts((current) => ({
                                  ...current,
                                  [conversation.phone]: {
                                    state: draft.state,
                                    note: event.target.value,
                                  },
                                }))
                              }
                            />
                            <button
                              className="button-secondary"
                              disabled={updateConversation.isPending}
                              onClick={() =>
                                updateConversation.mutate({
                                  phone: conversation.phone,
                                  state: draft.state,
                                  note: draft.note,
                                })
                              }
                              type="button"
                            >
                              Atualizar conversa
                            </button>
                          </div>
                        </div>
                      );
                    })
                  ) : (
                    <EmptyState label="Nenhuma conversa operacional ainda." />
                  )}
                </div>

                <div className="space-y-3">
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
                                instanceHeader,
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
          </div>

          <div className="grid gap-6">
            <FormPanel
              title="Fila e Observabilidade"
              icon={<LayoutList className="h-4 w-4" />}
              action={
                <button
                  className="button-secondary"
                  onClick={() =>
                    queryClient.invalidateQueries({
                      queryKey: QUERY_KEYS.queue(tenantHeader),
                    })
                  }
                  type="button"
                >
                  Atualizar fila
                </button>
              }
            >
              <div className="grid gap-3 sm:grid-cols-3">
                <QueueMetric label="Jobs" value={String(queueQuery.data?.jobs ?? 0)} />
                <QueueMetric
                  label="Dead letters"
                  value={String(queueQuery.data?.dead_letters ?? 0)}
                />
                <QueueMetric
                  label="Workers"
                  value={String(queueQuery.data?.workers ?? 0)}
                />
              </div>
              <div className="space-y-3">
                {(queueQuery.data?.recent ?? []).slice(0, 8).map((job) => (
                  <div
                    key={`${job.id}-${job.status}-${job.updated_at}`}
                    className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <p className="text-sm font-semibold text-white">
                          {job.kind}
                        </p>
                        <p className="mt-1 text-xs text-slate-400">
                          tentativa {job.attempt} • {formatDate(job.updated_at)}
                        </p>
                      </div>
                      <span className="badge">{job.status}</span>
                    </div>
                    {job.error ? (
                      <p className="mt-3 text-xs text-danger">{job.error}</p>
                    ) : null}
                  </div>
                ))}
                {!queueQuery.data?.recent?.length ? (
                  <EmptyState label="Fila sem atividade recente." />
                ) : null}
              </div>
              <div className="mt-4 space-y-3">
                <p className="text-sm font-semibold text-white">DLQ operacional</p>
                {queueDeadLettersQuery.data?.length ? (
                  queueDeadLettersQuery.data.slice(0, 8).map((job) => (
                    <div
                      key={`${job.id}-${job.updated_at}`}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold text-white">{job.kind}</p>
                          <p className="mt-1 text-xs text-slate-400">
                            tentativa {job.attempt} • {formatDate(job.updated_at)}
                          </p>
                        </div>
                        <button
                          className="button-secondary"
                          disabled={requeueDeadLetter.isPending}
                          onClick={() => requeueDeadLetter.mutate(job.id)}
                          type="button"
                        >
                          Requeue
                        </button>
                      </div>
                      <p className="mt-3 text-xs text-slate-500">{job.id}</p>
                      {job.error ? (
                        <p className="mt-2 text-xs text-danger">{job.error}</p>
                      ) : null}
                    </div>
                  ))
                ) : (
                  <EmptyState label="Nenhum job em dead-letter." />
                )}
              </div>
            </FormPanel>

            <FormPanel
              title="Campanhas"
              icon={<CirclePlay className="h-4 w-4" />}
              action={
                <button
                  className="button-primary"
                  disabled={
                    createCampaign.isPending ||
                    !campaignForm.name.trim() ||
                    !campaignForm.message.trim() ||
                    parsedContactPhones.length === 0
                  }
                  onClick={() => createCampaign.mutate()}
                  type="button"
                >
                  Criar campanha
                </button>
              }
            >
              <input
                className="input"
                placeholder="Nome da campanha"
                value={campaignForm.name}
                onChange={(event) =>
                  setCampaignForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
              <textarea
                className="input min-h-28 py-3"
                placeholder="Mensagem com variaveis, ex: Ola {{name}}"
                value={campaignForm.message}
                onChange={(event) =>
                  setCampaignForm((current) => ({
                    ...current,
                    message: event.target.value,
                  }))
                }
              />
              <input
                className="input"
                type="datetime-local"
                value={campaignForm.scheduledAt}
                onChange={(event) =>
                  setCampaignForm((current) => ({
                    ...current,
                    scheduledAt: event.target.value,
                  }))
                }
              />
              <div className="space-y-3">
                {campaignsQuery.data?.length ? (
                  campaignsQuery.data.map((campaign) => (
                    <div
                      key={campaign.id}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold text-white">
                            {campaign.name}
                          </p>
                          <p className="mt-1 text-xs text-slate-400">
                            {campaign.status} • {campaign.sent_count}/
                            {campaign.total_contacts} enviados
                          </p>
                        </div>
                        <button
                          className="button-secondary"
                          disabled={
                            runCampaign.isPending ||
                            campaign.status === "running"
                          }
                          onClick={() => runCampaign.mutate(campaign.id)}
                          type="button"
                        >
                          Rodar
                        </button>
                      </div>
                      <p className="mt-3 text-sm text-slate-300">
                        {campaign.message}
                      </p>
                    </div>
                  ))
                ) : (
                  <EmptyState label="Sem campanhas para esta instancia." />
                )}
              </div>
            </FormPanel>

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
                              .deleteWebhook(
                                token,
                                tenantHeader,
                                instanceHeader,
                                webhook.id,
                              )
                              .then(() => {
                                notify("Webhook removido.");
                                queryClient.invalidateQueries({
                                  queryKey: QUERY_KEYS.webhooks(
                                    tenantHeader,
                                    instanceHeader,
                                  ),
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
              <div className="mt-4 space-y-3">
                <p className="text-sm font-semibold text-white">
                  Entregas recentes
                </p>
                {webhookDeliveriesQuery.data?.length ? (
                  webhookDeliveriesQuery.data.slice(0, 8).map((delivery) => (
                    <div
                      key={delivery.id}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold text-white">
                            {delivery.event_type}
                          </p>
                          <p className="mt-1 text-xs text-slate-400">
                            {delivery.status} • tentativa {delivery.attempts} •{" "}
                            {formatDate(delivery.last_attempt_at || delivery.created_at)}
                          </p>
                        </div>
                        <button
                          className="button-secondary"
                          disabled={replayWebhookDelivery.isPending}
                          onClick={() => replayWebhookDelivery.mutate(delivery.id)}
                          type="button"
                        >
                          Replay
                        </button>
                      </div>
                      <p className="mt-3 text-xs text-slate-400">
                        webhook: {delivery.webhook_url}
                      </p>
                      {delivery.response_status ? (
                        <p className="mt-2 text-xs text-slate-500">
                          status HTTP: {delivery.response_status}
                        </p>
                      ) : null}
                      {delivery.last_error ? (
                        <p className="mt-2 text-xs text-danger">
                          {delivery.last_error}
                        </p>
                      ) : null}
                    </div>
                  ))
                ) : (
                  <EmptyState label="Sem histórico de entregas ainda." />
                )}
              </div>
            </FormPanel>

            <FormPanel
              title="Recursos de Dev"
              icon={<FileCode2 className="h-4 w-4" />}
              action={
                <a
                  className="button-primary"
                  href={makeAPIDocURL("openapi.yaml")}
                  target="_blank"
                >
                  Abrir OpenAPI
                </a>
              }
            >
              <a
                className="flex items-center justify-between rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-4 text-sm text-slate-300"
                href={makeAPIDocURL("openapi.yaml")}
                target="_blank"
              >
                <span>Especificacao OpenAPI</span>
                <FileCode2 className="h-4 w-4 text-glow" />
              </a>
              <a
                className="flex items-center justify-between rounded-2xl border border-white/10 bg-slate-950/60 px-4 py-4 text-sm text-slate-300"
                href={makeAPIDocURL("postman_collection.json")}
                target="_blank"
              >
                <span>Colecao Postman</span>
                <Files className="h-4 w-4 text-glow" />
              </a>
              <p className="text-sm leading-6 text-slate-400">
                Use os headers <code className="rounded bg-white/5 px-1 py-0.5">X-Tenant-ID</code> e{" "}
                <code className="rounded bg-white/5 px-1 py-0.5">X-Instance-ID</code> para testar fluxos multi-instancia.
              </p>
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
              title="Auditoria"
              icon={<ShieldCheck className="h-4 w-4" />}
              action={
                <button
                  className="button-secondary"
                  onClick={() =>
                    queryClient.invalidateQueries({
                      queryKey: QUERY_KEYS.audit(tenantHeader, instanceHeader),
                    })
                  }
                  type="button"
                >
                  Atualizar auditoria
                </button>
              }
            >
              <div className="space-y-3">
                {auditQuery.data?.length ? (
                  auditQuery.data.slice(0, 10).map((item) => (
                    <div
                      key={item.id}
                      className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <div>
                          <p className="text-sm font-semibold text-white">
                            {item.action}
                          </p>
                          <p className="mt-1 text-xs text-slate-400">
                            {item.resource} • {formatDate(item.created_at)}
                          </p>
                        </div>
                        <span className="badge">
                          {item.user_id ? item.user_id.slice(0, 8) : "system"}
                        </span>
                      </div>
                      {item.request_id ? (
                        <p className="mt-3 text-xs text-slate-500">
                          request: {item.request_id}
                        </p>
                      ) : null}
                    </div>
                  ))
                ) : (
                  <EmptyState label="Sem eventos de auditoria ainda." />
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
  instanceID: string,
  queryClient: ReturnType<typeof useQueryClient>,
) {
  if (!tenantID) return;
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.summary(tenantID) });
  if (instanceID) {
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.status(tenantID, instanceID),
    });
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.messages(tenantID, instanceID),
    });
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.conversations(tenantID, instanceID),
    });
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.groups(tenantID, instanceID),
    });
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.webhooks(tenantID, instanceID),
    });
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.webhookDeliveries(tenantID, instanceID),
    });
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.campaigns(tenantID, instanceID),
    });
  }
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.instances(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.apiKeys(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.queue(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.queueDeadLetters(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.usage(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members(tenantID) });
  if (instanceID) {
    queryClient.invalidateQueries({ queryKey: QUERY_KEYS.audit(tenantID, instanceID) });
  }
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

function QueueMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-slate-950/60 p-4">
      <p className="text-xs uppercase tracking-[0.22em] text-slate-500">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-white">{value}</p>
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

function parsePhonesFromCSV(content: string) {
  return extractPhonesFromText(content);
}

function extractPhonesFromText(content: string) {
  const phones = new Set<string>();
  const matches = content.match(PHONE_PATTERN) ?? [];
  for (const match of matches) {
    const trimmed = match.trim();
    if (trimmed) {
      phones.add(trimmed);
    }
  }
  return Array.from(phones);
}

function extractErrorMessage(error: unknown) {
  if (!(error instanceof Error) || !error.message.trim()) {
    return "Falha inesperada.";
  }
  return error.message.trim();
}

function buildInteractivePayload(form: {
  phone: string;
  type: string;
  header: string;
  body: string;
  footer: string;
  buttonText: string;
  buttonsText: string;
  listSectionsText: string;
  pollOptionsText: string;
  maxSelect: number;
}) {
  if (form.type === "buttons") {
    return {
      phone: form.phone,
      type: form.type,
      header: form.header,
      body: form.body,
      footer: form.footer,
      buttons: form.buttonsText
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter(Boolean)
        .map((line, index) => {
          const [idPart, titlePart] = line.split(":");
          return {
            id: idPart?.trim() || `btn_${index + 1}`,
            title: (titlePart ?? idPart ?? "").trim(),
          };
        })
        .filter((item) => item.title),
    };
  }

  if (form.type === "list") {
    return {
      phone: form.phone,
      type: form.type,
      header: form.header,
      body: form.body,
      footer: form.footer,
      button_text: form.buttonText,
      sections: form.listSectionsText
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter(Boolean)
        .map((line) => {
          const [title, rowsRaw = ""] = line.split("|");
          return {
            title: title.trim(),
            rows: rowsRaw
              .split(",")
              .map((row) => row.trim())
              .filter(Boolean)
              .map((row, index) => {
                const [id, rowTitle, description = ""] = row.split(":");
                return {
                  id: id?.trim() || `row_${index + 1}`,
                  title: (rowTitle ?? id ?? "").trim(),
                  description: description.trim(),
                };
              })
              .filter((row) => row.title),
          };
        })
        .filter((section) => section.title && section.rows.length),
    };
  }

  return {
    phone: form.phone,
    type: form.type,
    body: form.body,
    options: form.pollOptionsText
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean),
    max_select: form.maxSelect,
  };
}
