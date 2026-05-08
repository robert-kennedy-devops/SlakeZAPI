"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Ban,
  Cable,
  CirclePlay,
  Copy,
  Edit2,
  FileCode2,
  Files,
  Forward,
  LayoutList,
  KeyRound,
  Link,
  LoaderCircle,
  LogOut,
  MapPin,
  MessageSquare,
  MessageSquareShare,
  MessagesSquare,
  RadioTower,
  PlugZap,
  RefreshCcw,
  Send,
  Settings,
  ShieldCheck,
  Smile,
  Smartphone,
  SquareMousePointer,
  Star,
  Trash2,
  UserPlus,
  UsersRound,
  Webhook,
} from "lucide-react";
import { useRouter } from "next/navigation";
import type { ChangeEvent } from "react";
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
  CurrentUserResponse,
  Message,
  ResolvedContact,
  UserRole,
  WAContact,
} from "@/lib/types";
import { formatDate } from "@/lib/utils";
import { AutomationModule } from "@/components/dashboard/automation-module";
import {
  DashboardHeader,
  type DashboardExecutiveStat,
} from "@/components/dashboard/dashboard-header";
import { OperationsModule } from "@/components/dashboard/operations-module";
import {
  OverviewModule,
  type DashboardNavLink,
} from "@/components/dashboard/overview-module";
import { SettingsModule } from "@/components/dashboard/settings-module";
import {
  EmptyState,
  FormPanel,
  LoadingScreen,
  MemberRow,
  MetricCard,
  QueueMetric,
} from "@/components/dashboard/shared";

const QUERY_KEYS = {
  me: ["me"],
  summary: (tenantID: string) => ["summary", tenantID],
  instances: (tenantID: string) => ["instances", tenantID],
  campaigns: (tenantID: string, instanceID: string) => [
    "campaigns",
    tenantID,
    instanceID,
  ],
  status: (tenantID: string, instanceID: string) => [
    "status",
    tenantID,
    instanceID,
  ],
  messages: (tenantID: string, instanceID: string) => [
    "messages",
    tenantID,
    instanceID,
  ],
  conversations: (tenantID: string, instanceID: string) => [
    "conversations",
    tenantID,
    instanceID,
  ],
  queue: (tenantID: string) => ["queue", tenantID],
  queueDeadLetters: (tenantID: string) => ["queue-dead-letters", tenantID],
  groups: (tenantID: string, instanceID: string) => [
    "groups",
    tenantID,
    instanceID,
  ],
  webhooks: (tenantID: string, instanceID: string) => [
    "webhooks",
    tenantID,
    instanceID,
  ],
  webhookDeliveries: (tenantID: string, instanceID: string) => [
    "webhook-deliveries",
    tenantID,
    instanceID,
  ],
  apiKeys: (tenantID: string) => ["apikeys", tenantID],
  usage: (tenantID: string) => ["usage", tenantID],
  members: (tenantID: string) => ["members", tenantID],
  audit: (tenantID: string, instanceID: string) => [
    "audit",
    tenantID,
    instanceID,
  ],
};

const ROLE_LABELS: Record<UserRole, string> = {
  owner: "Owner",
  admin: "Admin",
  operator: "Operator",
  viewer: "Viewer",
};

const PHONE_PATTERN = /(?:\+?\d[\d \t().-]{7,}\d)/g;

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
  const [groupMessage, setGroupMessage] = useState("");
  const [selectedGroupJIDs, setSelectedGroupJIDs] = useState<string[]>([]);
  const [groupSearch, setGroupSearch] = useState("");
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

  // ── New feature state ──────────────────────────────────────────────────────
  const [locationForm, setLocationForm] = useState({
    phone: "",
    latitude: "",
    longitude: "",
    name: "",
    address: "",
  });
  const [contactCardForm, setContactCardForm] = useState({
    phone: "",
    contacts: "",
  });
  const [stickerForm, setStickerForm] = useState({ phone: "", url: "" });
  const [quotedForm, setQuotedForm] = useState({
    phone: "",
    message: "",
    quoted_message_id: "",
  });
  const [reactForm, setReactForm] = useState({
    phone: "",
    message_id: "",
    emoji: "👍",
  });
  const [deleteForm, setDeleteForm] = useState({
    phone: "",
    message_id: "",
    for_everyone: true,
  });
  const [profileDescForm, setProfileDescForm] = useState("");
  const [privacyForm, setPrivacyForm] = useState<{
    last_seen: string;
    profile_photo: string;
    status: string;
    read_receipts: boolean;
    group_add: string;
  }>({
    last_seen: "contacts",
    profile_photo: "contacts",
    status: "contacts",
    read_receipts: true,
    group_add: "contacts",
  });
  const [privacyLoaded, setPrivacyLoaded] = useState(false);
  const [createGroupForm, setCreateGroupForm] = useState({
    name: "",
    participants: "",
  });
  const [groupParticipantsForm, setGroupParticipantsForm] = useState({
    jid: "",
    participants: "",
    action: "add" as "add" | "remove" | "promote" | "demote",
  });
  const [groupInfoForm, setGroupInfoForm] = useState({
    jid: "",
    name: "",
    description: "",
  });
  const [leaveGroupJID, setLeaveGroupJID] = useState("");
  const [inviteLinkJID, setInviteLinkJID] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [blockPhone, setBlockPhone] = useState("");
  const [chatForm, setChatForm] = useState({
    phone: "",
    action: "archive" as
      | "archive"
      | "unarchive"
      | "mute"
      | "unmute"
      | "pin"
      | "unpin"
      | "read"
      | "unread",
    mute_hours: "8",
  });
  const [editForm, setEditForm] = useState({
    phone: "",
    message_id: "",
    new_message: "",
  });
  const [forwardForm, setForwardForm] = useState({ phone: "", message: "" });
  const [starForm, setStarForm] = useState({
    phone: "",
    message_id: "",
    starred: true,
    from_me: true,
  });
  const [pairPhoneForm, setPairPhoneForm] = useState({ phone: "" });
  const [pairCode, setPairCode] = useState("");

  useEffect(() => {
    const authToken = getAuthToken();
    if (authToken) {
      setToken(authToken);
      setTenant(getSelectedTenantID());
      return;
    }
    // No in-memory token (page reload) — recover session via HttpOnly refresh cookie.
    api
      .refresh(getSelectedTenantID() || undefined)
      .then((session) => {
        setToken(session.token);
        setTenant(session.tenant?.id ?? getSelectedTenantID());
        if (session.tenant?.id) setSelectedTenantID(session.tenant.id);
      })
      .catch(() => {
        router.replace("/login");
      });
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
    const socket = new WebSocket(makeWSURL(tenantHeader, instanceHeader));
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
    mutationFn: () =>
      api.sendMessage(token, tenantHeader, instanceHeader, sendForm),
    onSuccess: () => {
      setSendForm({ phone: "", message: "" });
      notify("Mensagem enviada.");
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
    },
    onError: handleMutationError,
  });

  const sendMedia = useMutation({
    mutationFn: () =>
      api.sendMedia(token, tenantHeader, instanceHeader, mediaForm),
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
      Promise.allSettled(
        selectedGroupJIDs.map((jid) =>
          api.sendGroupMessage(token, tenantHeader, instanceHeader, {
            group_jid: jid,
            message: groupMessage,
          }),
        ),
      ),
    onSuccess: (results) => {
      const ok = results.filter((r) => r.status === "fulfilled").length;
      const fail = results.length - ok;
      setSelectedGroupJIDs([]);
      setGroupMessage("");
      notify(
        fail === 0
          ? `Mensagem enviada para ${ok} grupo${ok !== 1 ? "s" : ""}.`
          : `${ok} enviado${ok !== 1 ? "s" : ""}, ${fail} falhou.`,
      );
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

  const importWAContacts = useMutation({
    mutationFn: () => api.waContacts(token, tenantHeader, instanceHeader),
    onSuccess: (contacts: WAContact[]) => {
      if (!contacts.length) {
        notify("Nenhum contato encontrado no cache do WhatsApp.");
        return;
      }
      const phones = contacts.map((c) => c.phone).join("\n");
      setContactInput(phones);
      notify(`${contacts.length} contatos importados do WhatsApp.`);
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
    mutationFn: () =>
      api.createInstance(token, tenantHeader, { name: instanceName }),
    onSuccess: (instance) => {
      setInstanceName("");
      setSelectedInstanceID(instance.id);
      notify("Instancia criada.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.instances(tenantHeader),
      });
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
      api.replayWebhookDelivery(
        token,
        tenantHeader,
        instanceHeader,
        deliveryID,
      ),
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
    mutationFn: (jobID: string) =>
      api.requeueDeadLetter(token, tenantHeader, jobID),
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

  const profileQuery = useQuery({
    queryKey: ["profile", tenantHeader, instanceHeader],
    queryFn: () => api.whatsappProfile(token, tenantHeader, instanceHeader),
    enabled: !!token && !!tenantHeader,
  });

  const privacyQuery = useQuery({
    queryKey: ["privacy", tenantHeader, instanceHeader],
    queryFn: () => api.privacySettings(token, tenantHeader, instanceHeader),
    enabled: !!token && !!tenantHeader,
  });

  useEffect(() => {
    if (privacyQuery.data && !privacyLoaded) {
      const d = privacyQuery.data;
      setPrivacyForm({
        last_seen: d.last_seen,
        profile_photo: d.profile_photo,
        status: d.status,
        read_receipts: d.read_receipts,
        group_add: d.group_add,
      });
      setPrivacyLoaded(true);
    }
  }, [privacyQuery.data, privacyLoaded]);

  const sendLocation = useMutation({
    mutationFn: () =>
      api.sendLocation(token, tenantHeader, instanceHeader, {
        phone: locationForm.phone,
        latitude: parseFloat(locationForm.latitude),
        longitude: parseFloat(locationForm.longitude),
        name: locationForm.name,
        address: locationForm.address,
      }),
    onSuccess: () => {
      setLocationForm({
        phone: "",
        latitude: "",
        longitude: "",
        name: "",
        address: "",
      });
      notify("Localização enviada.");
    },
    onError: handleMutationError,
  });

  const sendContactCard = useMutation({
    mutationFn: () =>
      api.sendContactCard(token, tenantHeader, instanceHeader, {
        phone: contactCardForm.phone,
        contacts: contactCardForm.contacts
          .split(",")
          .map((p) => p.trim())
          .filter(Boolean),
      }),
    onSuccess: () => {
      setContactCardForm({ phone: "", contacts: "" });
      notify("Contato(s) enviado(s).");
    },
    onError: handleMutationError,
  });

  const sendSticker = useMutation({
    mutationFn: () =>
      api.sendSticker(token, tenantHeader, instanceHeader, {
        phone: stickerForm.phone,
        url: stickerForm.url,
      }),
    onSuccess: () => {
      setStickerForm({ phone: "", url: "" });
      notify("Sticker enviado.");
    },
    onError: handleMutationError,
  });

  const sendQuoted = useMutation({
    mutationFn: () =>
      api.sendQuoted(token, tenantHeader, instanceHeader, {
        phone: quotedForm.phone,
        message: quotedForm.message,
        quoted_message_id: quotedForm.quoted_message_id,
      }),
    onSuccess: () => {
      setQuotedForm({ phone: "", message: "", quoted_message_id: "" });
      notify("Resposta enviada.");
    },
    onError: handleMutationError,
  });

  const reactToMessage = useMutation({
    mutationFn: () =>
      api.reactToMessage(token, tenantHeader, instanceHeader, {
        phone: reactForm.phone,
        message_id: reactForm.message_id,
        emoji: reactForm.emoji,
      }),
    onSuccess: () => {
      setReactForm({ phone: "", message_id: "", emoji: "👍" });
      notify("Reação enviada.");
    },
    onError: handleMutationError,
  });

  const deleteMessage = useMutation({
    mutationFn: () =>
      api.deleteMessage(token, tenantHeader, instanceHeader, {
        phone: deleteForm.phone,
        message_id: deleteForm.message_id,
        for_everyone: deleteForm.for_everyone,
      }),
    onSuccess: () => {
      setDeleteForm({ phone: "", message_id: "", for_everyone: true });
      notify("Mensagem apagada.");
    },
    onError: handleMutationError,
  });

  const updateProfile = useMutation({
    mutationFn: () =>
      api.updateWhatsappProfile(token, tenantHeader, instanceHeader, {
        description: profileDescForm,
      }),
    onSuccess: () => {
      notify("Perfil atualizado.");
      queryClient.invalidateQueries({
        queryKey: ["profile", tenantHeader, instanceHeader],
      });
    },
    onError: handleMutationError,
  });

  const updatePrivacy = useMutation({
    mutationFn: () =>
      api.updatePrivacySettings(
        token,
        tenantHeader,
        instanceHeader,
        privacyForm,
      ),
    onSuccess: () => {
      notify("Privacidade atualizada.");
      queryClient.invalidateQueries({
        queryKey: ["privacy", tenantHeader, instanceHeader],
      });
    },
    onError: handleMutationError,
  });

  const createGroup = useMutation({
    mutationFn: () =>
      api.createGroup(token, tenantHeader, instanceHeader, {
        name: createGroupForm.name,
        participants: createGroupForm.participants
          .split(",")
          .map((p) => p.trim())
          .filter(Boolean),
      }),
    onSuccess: () => {
      setCreateGroupForm({ name: "", participants: "" });
      notify("Grupo criado.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.groups(tenantHeader, instanceHeader),
      });
    },
    onError: handleMutationError,
  });

  const updateGroupParticipants = useMutation({
    mutationFn: () =>
      api.updateGroupParticipants(
        token,
        tenantHeader,
        instanceHeader,
        groupParticipantsForm.jid,
        {
          participants: groupParticipantsForm.participants
            .split(",")
            .map((p) => p.trim())
            .filter(Boolean),
          action: groupParticipantsForm.action,
        },
      ),
    onSuccess: () => {
      setGroupParticipantsForm({ jid: "", participants: "", action: "add" });
      notify("Participantes atualizados.");
    },
    onError: handleMutationError,
  });

  const updateGroupInfo = useMutation({
    mutationFn: () =>
      api.updateGroupInfo(
        token,
        tenantHeader,
        instanceHeader,
        groupInfoForm.jid,
        {
          name: groupInfoForm.name || undefined,
          description: groupInfoForm.description || undefined,
        },
      ),
    onSuccess: () => {
      setGroupInfoForm({ jid: "", name: "", description: "" });
      notify("Grupo atualizado.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.groups(tenantHeader, instanceHeader),
      });
    },
    onError: handleMutationError,
  });

  const leaveGroup = useMutation({
    mutationFn: () =>
      api.leaveGroup(token, tenantHeader, instanceHeader, leaveGroupJID),
    onSuccess: () => {
      setLeaveGroupJID("");
      notify("Saiu do grupo.");
      queryClient.invalidateQueries({
        queryKey: QUERY_KEYS.groups(tenantHeader, instanceHeader),
      });
    },
    onError: handleMutationError,
  });

  const fetchInviteLink = useMutation({
    mutationFn: () =>
      api.groupInviteLink(token, tenantHeader, instanceHeader, inviteLinkJID),
    onSuccess: (data) => {
      setInviteLink(data.invite_link);
      notify("Link de convite obtido.");
    },
    onError: handleMutationError,
  });

  const blockContact = useMutation({
    mutationFn: (action: "block" | "unblock") =>
      action === "block"
        ? api.blockContact(token, tenantHeader, instanceHeader, blockPhone)
        : api.unblockContact(token, tenantHeader, instanceHeader, blockPhone),
    onSuccess: (_, action) => {
      setBlockPhone("");
      notify(
        action === "block" ? "Contato bloqueado." : "Contato desbloqueado.",
      );
    },
    onError: handleMutationError,
  });

  const chatAction = useMutation({
    mutationFn: () => {
      const phone = chatForm.phone;
      switch (chatForm.action) {
        case "archive":
          return api.archiveChat(token, tenantHeader, phone, {
            phone,
            archive: true,
          });
        case "unarchive":
          return api.archiveChat(token, tenantHeader, phone, {
            phone,
            archive: false,
          });
        case "mute":
          return api.muteChat(token, tenantHeader, phone, {
            phone,
            mute: true,
            duration_hours: parseInt(chatForm.mute_hours) || 0,
          });
        case "unmute":
          return api.muteChat(token, tenantHeader, phone, {
            phone,
            mute: false,
          });
        case "pin":
          return api.pinChat(token, tenantHeader, phone, { phone, pin: true });
        case "unpin":
          return api.pinChat(token, tenantHeader, phone, { phone, pin: false });
        case "read":
          return api.markChatRead(token, tenantHeader, phone, {
            phone,
            read: true,
          });
        case "unread":
          return api.markChatRead(token, tenantHeader, phone, {
            phone,
            read: false,
          });
      }
    },
    onSuccess: () => {
      notify("Acao de chat aplicada.");
    },
    onError: handleMutationError,
  });

  const editMessage = useMutation({
    mutationFn: () => api.editMessage(token, tenantHeader, editForm),
    onSuccess: () => {
      setEditForm({ phone: "", message_id: "", new_message: "" });
      notify("Mensagem editada.");
    },
    onError: handleMutationError,
  });

  const forwardMessage = useMutation({
    mutationFn: () => api.forwardMessage(token, tenantHeader, forwardForm),
    onSuccess: () => {
      setForwardForm({ phone: "", message: "" });
      notify("Mensagem encaminhada.");
    },
    onError: handleMutationError,
  });

  const starMessage = useMutation({
    mutationFn: () => api.starMessage(token, tenantHeader, starForm),
    onSuccess: () => {
      notify(
        starForm.starred
          ? "Mensagem marcada com estrela."
          : "Estrela removida.",
      );
    },
    onError: handleMutationError,
  });

  const pairPhone = useMutation({
    mutationFn: () =>
      api.pairPhone(token, tenantHeader, { phone: pairPhoneForm.phone }),
    onSuccess: (data) => {
      setPairCode(data.code);
      notify("Codigo de pareamento gerado!");
    },
    onError: handleMutationError,
  });

  const restartInstance = useMutation({
    mutationFn: () => api.restartInstance(token, tenantHeader),
    onSuccess: () => {
      invalidateTenant(tenantHeader, instanceHeader, queryClient);
      notify("Instancia reiniciada.");
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
  const unreadConversations =
    conversationsQuery.data?.reduce(
      (total, conversation) => total + conversation.unread_count,
      0,
    ) ?? 0;
  const connectedInstances =
    instancesQuery.data?.filter((instance) => instance.status === "connected")
      .length ?? 0;
  const activeCampaigns =
    campaignsQuery.data?.filter(
      (campaign) =>
        campaign.status === "running" || campaign.status === "scheduled",
    ).length ?? 0;
  const sectionLinks: DashboardNavLink[] = [
    {
      id: "overview",
      label: "Visao Geral",
      icon: <Activity className="h-4 w-4" />,
    },
    {
      id: "operations",
      label: "Operacao",
      icon: <MessageSquare className="h-4 w-4" />,
    },
    {
      id: "advanced",
      label: "Recursos Avancados",
      icon: <MessagesSquare className="h-4 w-4" />,
    },
    {
      id: "identity",
      label: "Conta e Privacidade",
      icon: <ShieldCheck className="h-4 w-4" />,
    },
    {
      id: "tools",
      label: "Ferramentas de Chat",
      icon: <Settings className="h-4 w-4" />,
    },
  ];
  const executiveStats: DashboardExecutiveStat[] = [
    {
      label: "Instancias conectadas",
      value: `${connectedInstances}/${instancesQuery.data?.length ?? 0}`,
      hint: connectedInstances > 0 ? "operacao ativa" : "aguardando conexao",
    },
    {
      label: "Conversas pendentes",
      value: String(unreadConversations),
      hint: unreadConversations > 0 ? "demandam atencao" : "fila sob controle",
    },
    {
      label: "Campanhas em curso",
      value: String(activeCampaigns),
      hint: activeCampaigns > 0 ? "execucao em andamento" : "sem agendamentos",
    },
    {
      label: "Webhooks ativos",
      value: String(webhooksQuery.data?.length ?? 0),
      hint: "integracoes configuradas",
    },
  ];

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
      <div className="hero-orb left-[-140px] top-28 h-80 w-80 bg-glow/15" />
      <div className="hero-orb right-[-120px] top-[30rem] h-72 w-72 bg-neon/10" />
      <div className="grid-background absolute inset-0 opacity-35" />
      <div className="relative mx-auto flex max-w-7xl flex-col gap-6">
        <DashboardHeader
          title={
            summaryQuery.data?.tenant?.name ??
            currentUser?.tenant?.name ??
            "Console"
          }
          description="Um dashboard comercialmente forte precisa comunicar controle, confianca e clareza. Esta organizacao prioriza saude da operacao, produtividade da equipe e acesso gradual aos recursos avancados."
          operatorName={currentUser?.user.name}
          operatorRole={ROLE_LABELS[currentMembership?.role ?? "viewer"]}
          badges={[
            `tenant ${tenantHeader ? tenantHeader.slice(0, 8) : "n/a"}`,
            `instancia ${instanceHeader ? instanceHeader.slice(0, 8) : "n/a"}`,
            statusQuery.data?.status ?? "disconnected",
          ]}
          executiveStats={executiveStats}
          controls={
            <div className="rounded-3xl border border-white/10 bg-slate-950/45 p-4">
              <div className="mb-4 rounded-3xl border border-glow/20 bg-glow/10 p-4">
                <p className="text-xs uppercase tracking-[0.24em] text-glow">
                  Proximos passos recomendados
                </p>
                <ol className="mt-3 space-y-2 text-sm text-slate-200">
                  <li>1. Selecione o workspace e a instancia ativa.</li>
                  <li>2. Conecte o WhatsApp e valide o QR ou codigo.</li>
                  <li>
                    3. Envie a primeira mensagem e acompanhe inbox e campanhas.
                  </li>
                </ol>
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <select
                  className="input"
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
                  className="input"
                  value={instanceHeader}
                  onChange={(event) =>
                    setSelectedInstanceID(event.target.value)
                  }
                >
                  {instancesQuery.data?.map((instance) => (
                    <option key={instance.id} value={instance.id}>
                      {instance.name} • {instance.phone || instance.status}
                    </option>
                  ))}
                </select>
              </div>
              <div className="mt-3 flex flex-col gap-3 sm:flex-row">
                <input
                  className="input"
                  placeholder="Nome da nova instancia"
                  value={instanceName}
                  onChange={(event) => setInstanceName(event.target.value)}
                />
                <button
                  className="button-secondary sm:min-w-48"
                  disabled={createInstance.isPending || !instanceName.trim()}
                  onClick={() => createInstance.mutate()}
                  type="button"
                >
                  <Smartphone className="mr-2 h-4 w-4" />
                  Criar instancia
                </button>
              </div>
              <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:justify-end">
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
                <button
                  className="button-danger"
                  onClick={signOut}
                  type="button"
                >
                  <LogOut className="mr-2 h-4 w-4" />
                  Sair
                </button>
              </div>
            </div>
          }
        />

        {flash ? (
          <div className="rounded-2xl border border-glow/20 bg-glow/10 px-4 py-3 text-sm text-glow">
            {flash}
          </div>
        ) : null}

        <OverviewModule
          links={sectionLinks}
          metrics={
            <>
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
            </>
          }
        />

        <OperationsModule
          primary={
            <>
              <div className="panel p-6">
                <div className="flex flex-wrap items-center justify-between gap-4">
                  <div>
                    <p className="section-kicker">Operacao principal</p>
                    <h2 className="mt-2 text-xl font-semibold text-white">
                      Pareamento, status e QR
                    </h2>
                    <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                      Esta area concentra as acoes mais valiosas para
                      demonstracao: conectar a instancia, validar uso do plano e
                      disparar as primeiras mensagens.
                    </p>
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
                          // eslint-disable-next-line no-inline-styles
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
                          src={makeQRImageURL(tenantHeader, instanceHeader)}
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

              <div className="grid gap-6 xl:grid-cols-2 2xl:grid-cols-3">
                <FormPanel
                  title="Mensagens Interativas"
                  icon={<SquareMousePointer className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        sendInteractive.isPending ||
                        !interactiveForm.body.trim()
                      }
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
                      disabled={
                        sendGroup.isPending ||
                        selectedGroupJIDs.length === 0 ||
                        !groupMessage.trim()
                      }
                      onClick={() => sendGroup.mutate()}
                      type="button"
                    >
                      {sendGroup.isPending
                        ? "Enviando..."
                        : `Enviar${selectedGroupJIDs.length > 1 ? ` (${selectedGroupJIDs.length})` : ""}`}
                    </button>
                  }
                >
                  <textarea
                    className="input min-h-24 py-3"
                    placeholder="Mensagem para o(s) grupo(s)"
                    value={groupMessage}
                    onChange={(event) => setGroupMessage(event.target.value)}
                  />
                  <div className="space-y-1.5">
                    <input
                      className="input w-full"
                      placeholder="Buscar grupo..."
                      value={groupSearch}
                      onChange={(e) => setGroupSearch(e.target.value)}
                    />
                    <div className="flex items-center justify-between px-1 text-xs text-slate-500">
                      <span>
                        {groupsQuery.data
                          ? `${groupsQuery.data.filter((g) => g.name.toLowerCase().includes(groupSearch.toLowerCase())).length} grupo(s)`
                          : "—"}
                      </span>
                      {selectedGroupJIDs.length > 0 && (
                        <button
                          type="button"
                          className="text-indigo-400 hover:text-white transition-colors"
                          onClick={() => setSelectedGroupJIDs([])}
                        >
                          {selectedGroupJIDs.length} selecionado
                          {selectedGroupJIDs.length !== 1 ? "s" : ""} · Limpar
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="max-h-80 space-y-1.5 overflow-y-auto pr-1">
                    {groupsQuery.data
                      ?.filter((g) =>
                        g.name
                          .toLowerCase()
                          .includes(groupSearch.toLowerCase()),
                      )
                      .map((group) => {
                        const selected = selectedGroupJIDs.includes(group.jid);
                        return (
                          <button
                            key={group.jid}
                            type="button"
                            className={`flex w-full items-start gap-3 rounded-xl border px-3 py-2.5 text-left text-sm transition-colors ${
                              selected
                                ? "border-indigo-500/50 bg-indigo-900/25 text-white"
                                : "border-white/8 bg-slate-950/40 text-slate-300 hover:border-white/20 hover:bg-slate-900/60"
                            }`}
                            onClick={() =>
                              setSelectedGroupJIDs((current) =>
                                current.includes(group.jid)
                                  ? current.filter((j) => j !== group.jid)
                                  : [...current, group.jid],
                              )
                            }
                          >
                            <span
                              className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-md border text-xs font-bold transition-colors ${
                                selected
                                  ? "border-indigo-400 bg-indigo-500 text-white"
                                  : "border-slate-600 bg-transparent text-transparent"
                              }`}
                            >
                              ✓
                            </span>
                            <span className="min-w-0 flex-1">
                              <span className="block truncate leading-snug">
                                {group.name}
                              </span>
                              <span className="mt-0.5 block text-xs text-slate-500">
                                {group.participant_count > 0
                                  ? `${group.participant_count} membros`
                                  : "—"}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    {groupsQuery.isLoading ? (
                      <EmptyState label="Carregando grupos da instancia..." />
                    ) : null}
                    {!groupsQuery.isLoading && groupsErrorMessage ? (
                      <EmptyState
                        label={`Grupos indisponiveis: ${groupsErrorMessage}`}
                      />
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
                    <p className="panel-subtitle max-w-2xl">
                      Cole uma lista de numeros com DDI. O backend valida quais
                      existem no WhatsApp e voce pode selecionar varios para
                      disparo em lote.
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    <button
                      type="button"
                      className="button-secondary"
                      disabled={importWAContacts.isPending}
                      onClick={() => importWAContacts.mutate()}
                      title="Importa os contatos salvos no cache local do WhatsApp"
                    >
                      {importWAContacts.isPending
                        ? "Importando..."
                        : "Importar do WhatsApp"}
                    </button>
                    <label
                      className="button-secondary cursor-pointer"
                      htmlFor={csvInputID}
                    >
                      Importar CSV
                    </label>
                    <button
                      className="button-secondary"
                      disabled={
                        resolveContacts.isPending ||
                        parsedContactPhones.length === 0
                      }
                      onClick={() => {
                        if (!parsedContactPhones.length) {
                          notify(
                            "Adicione pelo menos um telefone valido para reconhecer.",
                          );
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
                      <span>
                        {parsedContactPhones.length} numeros detectados
                      </span>
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
                          Contatos com WhatsApp podem ser selecionados para
                          envio em massa.
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

                    <div className="stack-scroll max-h-[32rem]">
                      {resolvedContacts.length ? (
                        resolvedContacts.map((contact) => {
                          const checked = selectedBulkPhones.includes(
                            contact.phone,
                          );
                          const disabled =
                            !contact.is_whatsapp || !contact.phone;
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
                  <div className="stack-scroll max-h-[42rem]">
                    {conversationsQuery.data?.length ? (
                      conversationsQuery.data
                        .slice(0, 8)
                        .map((conversation) => {
                          const draft = conversationDrafts[
                            conversation.phone
                          ] ?? {
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
                                {conversation.last_message_body ||
                                  "Sem resumo textual"}
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

                  <div className="stack-scroll max-h-[42rem]">
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
            </>
          }
          sidebar={
            <>
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
                  <QueueMetric
                    label="Jobs"
                    value={String(queueQuery.data?.jobs ?? 0)}
                  />
                  <QueueMetric
                    label="Dead letters"
                    value={String(queueQuery.data?.dead_letters ?? 0)}
                  />
                  <QueueMetric
                    label="Workers"
                    value={String(queueQuery.data?.workers ?? 0)}
                  />
                </div>
                <div className="stack-scroll max-h-[20rem]">
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
                            tentativa {job.attempt} •{" "}
                            {formatDate(job.updated_at)}
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
                <div className="mt-4">
                  <p className="text-sm font-semibold text-white">
                    DLQ operacional
                  </p>
                  <div className="stack-scroll mt-3 max-h-[20rem]">
                    {queueDeadLettersQuery.data?.length ? (
                      queueDeadLettersQuery.data.slice(0, 8).map((job) => (
                        <div
                          key={`${job.id}-${job.updated_at}`}
                          className="rounded-2xl border border-white/10 bg-slate-950/60 p-4"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <p className="text-sm font-semibold text-white">
                                {job.kind}
                              </p>
                              <p className="mt-1 text-xs text-slate-400">
                                tentativa {job.attempt} •{" "}
                                {formatDate(job.updated_at)}
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
                          <p className="mt-3 text-xs text-slate-500">
                            {job.id}
                          </p>
                          {job.error ? (
                            <p className="mt-2 text-xs text-danger">
                              {job.error}
                            </p>
                          ) : null}
                        </div>
                      ))
                    ) : (
                      <EmptyState label="Nenhum job em dead-letter." />
                    )}
                  </div>
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
                <div className="stack-scroll max-h-[24rem]">
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
                <div className="stack-scroll max-h-[20rem]">
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
                <div className="mt-4">
                  <p className="text-sm font-semibold text-white">
                    Entregas recentes
                  </p>
                  <div className="stack-scroll mt-3 max-h-[22rem]">
                    {webhookDeliveriesQuery.data?.length ? (
                      webhookDeliveriesQuery.data
                        .slice(0, 8)
                        .map((delivery) => (
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
                                  {delivery.status} • tentativa{" "}
                                  {delivery.attempts} •{" "}
                                  {formatDate(
                                    delivery.last_attempt_at ||
                                      delivery.created_at,
                                  )}
                                </p>
                              </div>
                              <button
                                className="button-secondary"
                                disabled={replayWebhookDelivery.isPending}
                                onClick={() =>
                                  replayWebhookDelivery.mutate(delivery.id)
                                }
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
                  Use os headers{" "}
                  <code className="rounded bg-white/5 px-1 py-0.5">
                    X-Tenant-ID
                  </code>{" "}
                  e{" "}
                  <code className="rounded bg-white/5 px-1 py-0.5">
                    X-Instance-ID
                  </code>{" "}
                  para testar fluxos multi-instancia.
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
                        onClick={() =>
                          navigator.clipboard.writeText(lastSecret)
                        }
                        type="button"
                      >
                        <Copy className="mr-2 h-4 w-4" />
                        Copiar
                      </button>
                    </div>
                  </div>
                ) : null}
                <div className="stack-scroll max-h-[20rem]">
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
                        queryKey: QUERY_KEYS.audit(
                          tenantHeader,
                          instanceHeader,
                        ),
                      })
                    }
                    type="button"
                  >
                    Atualizar auditoria
                  </button>
                }
              >
                <div className="stack-scroll max-h-[22rem]">
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
                <div className="stack-scroll max-h-[22rem]">
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
            </>
          }
        />

        {/* ── Novos tipos de mensagem ───────────────────────────────────── */}
        <AutomationModule
          advancedMessaging={
            <>
              <div className="panel p-6">
                <p className="section-kicker">Recursos avancados</p>
                <h2 className="mt-2 text-xl font-semibold text-white">
                  Localização, Contatos, Sticker e Respostas
                </h2>
                <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                  Funcoes de maior profundidade ficam agrupadas aqui para nao
                  competir com a jornada principal do usuario.
                </p>
              </div>

              <div className="grid gap-6 lg:grid-cols-2">
                <FormPanel
                  title="Enviar Localizacao"
                  icon={<MapPin className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        sendLocation.isPending ||
                        !locationForm.phone ||
                        !locationForm.latitude ||
                        !locationForm.longitude
                      }
                      onClick={() => sendLocation.mutate()}
                      type="button"
                    >
                      Enviar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={locationForm.phone}
                    onChange={(e) =>
                      setLocationForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <div className="grid grid-cols-2 gap-3">
                    <input
                      className="input"
                      placeholder="Latitude ex: -23.5505"
                      value={locationForm.latitude}
                      onChange={(e) =>
                        setLocationForm((f) => ({
                          ...f,
                          latitude: e.target.value,
                        }))
                      }
                    />
                    <input
                      className="input"
                      placeholder="Longitude ex: -46.6333"
                      value={locationForm.longitude}
                      onChange={(e) =>
                        setLocationForm((f) => ({
                          ...f,
                          longitude: e.target.value,
                        }))
                      }
                    />
                  </div>
                  <input
                    className="input"
                    placeholder="Nome do local (opcional)"
                    value={locationForm.name}
                    onChange={(e) =>
                      setLocationForm((f) => ({ ...f, name: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="Endereco (opcional)"
                    value={locationForm.address}
                    onChange={(e) =>
                      setLocationForm((f) => ({
                        ...f,
                        address: e.target.value,
                      }))
                    }
                  />
                </FormPanel>

                <FormPanel
                  title="Enviar Cartao de Contato"
                  icon={<UserPlus className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        sendContactCard.isPending ||
                        !contactCardForm.phone ||
                        !contactCardForm.contacts
                      }
                      onClick={() => sendContactCard.mutate()}
                      type="button"
                    >
                      Enviar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone destinatario (DDI+numero)"
                    value={contactCardForm.phone}
                    onChange={(e) =>
                      setContactCardForm((f) => ({
                        ...f,
                        phone: e.target.value,
                      }))
                    }
                  />
                  <textarea
                    className="input min-h-24 py-3"
                    placeholder={
                      "Contatos a compartilhar (separados por virgula):\n5511999990001, 5511999990002"
                    }
                    value={contactCardForm.contacts}
                    onChange={(e) =>
                      setContactCardForm((f) => ({
                        ...f,
                        contacts: e.target.value,
                      }))
                    }
                  />
                </FormPanel>

                <FormPanel
                  title="Enviar Sticker (WebP)"
                  icon={<Smile className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        sendSticker.isPending ||
                        !stickerForm.phone ||
                        !stickerForm.url
                      }
                      onClick={() => sendSticker.mutate()}
                      type="button"
                    >
                      Enviar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={stickerForm.phone}
                    onChange={(e) =>
                      setStickerForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="URL do arquivo .webp"
                    value={stickerForm.url}
                    onChange={(e) =>
                      setStickerForm((f) => ({ ...f, url: e.target.value }))
                    }
                  />
                </FormPanel>

                <FormPanel
                  title="Responder Mensagem"
                  icon={<MessageSquareShare className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        sendQuoted.isPending ||
                        !quotedForm.phone ||
                        !quotedForm.message ||
                        !quotedForm.quoted_message_id
                      }
                      onClick={() => sendQuoted.mutate()}
                      type="button"
                    >
                      Responder
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={quotedForm.phone}
                    onChange={(e) =>
                      setQuotedForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="ID da mensagem original (WhatsApp ID)"
                    value={quotedForm.quoted_message_id}
                    onChange={(e) =>
                      setQuotedForm((f) => ({
                        ...f,
                        quoted_message_id: e.target.value,
                      }))
                    }
                  />
                  <textarea
                    className="input min-h-24 py-3"
                    placeholder="Sua resposta..."
                    value={quotedForm.message}
                    onChange={(e) =>
                      setQuotedForm((f) => ({ ...f, message: e.target.value }))
                    }
                  />
                </FormPanel>

                <FormPanel
                  title="Reagir a Mensagem"
                  icon={<Smile className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        reactToMessage.isPending ||
                        !reactForm.phone ||
                        !reactForm.message_id
                      }
                      onClick={() => reactToMessage.mutate()}
                      type="button"
                    >
                      Reagir
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={reactForm.phone}
                    onChange={(e) =>
                      setReactForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="ID da mensagem (WhatsApp ID)"
                    value={reactForm.message_id}
                    onChange={(e) =>
                      setReactForm((f) => ({
                        ...f,
                        message_id: e.target.value,
                      }))
                    }
                  />
                  <div className="flex gap-2 flex-wrap">
                    {[
                      "👍",
                      "❤️",
                      "😂",
                      "😮",
                      "😢",
                      "🙏",
                      "🔥",
                      "🎉",
                      "👏",
                      "✅",
                    ].map((emoji) => (
                      <button
                        key={emoji}
                        type="button"
                        className={`text-xl px-2 py-1 rounded-xl border transition-colors ${reactForm.emoji === emoji ? "border-glow bg-glow/20" : "border-white/10 bg-white/5 hover:border-white/30"}`}
                        onClick={() => setReactForm((f) => ({ ...f, emoji }))}
                      >
                        {emoji}
                      </button>
                    ))}
                    <input
                      className="input flex-1 min-w-20"
                      placeholder="Outro emoji"
                      value={reactForm.emoji}
                      onChange={(e) =>
                        setReactForm((f) => ({ ...f, emoji: e.target.value }))
                      }
                    />
                  </div>
                </FormPanel>

                <FormPanel
                  title="Apagar Mensagem"
                  icon={<Trash2 className="h-4 w-4" />}
                  action={
                    <button
                      className="button-danger"
                      disabled={
                        deleteMessage.isPending ||
                        !deleteForm.phone ||
                        !deleteForm.message_id
                      }
                      onClick={() => deleteMessage.mutate()}
                      type="button"
                    >
                      Apagar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={deleteForm.phone}
                    onChange={(e) =>
                      setDeleteForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="ID da mensagem (WhatsApp ID)"
                    value={deleteForm.message_id}
                    onChange={(e) =>
                      setDeleteForm((f) => ({
                        ...f,
                        message_id: e.target.value,
                      }))
                    }
                  />
                  <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={deleteForm.for_everyone}
                      onChange={(e) =>
                        setDeleteForm((f) => ({
                          ...f,
                          for_everyone: e.target.checked,
                        }))
                      }
                      className="rounded"
                    />
                    Apagar para todos (revoke)
                  </label>
                </FormPanel>
              </div>
            </>
          }
          groupManagement={
            <>
              <div className="panel p-6">
                <p className="section-kicker">Gestao colaborativa</p>
                <h2 className="mt-2 text-xl font-semibold text-white">
                  Criar, editar e administrar grupos
                </h2>
              </div>

              <FormPanel
                title="Criar Grupo"
                icon={<UsersRound className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={createGroup.isPending || !createGroupForm.name}
                    onClick={() => createGroup.mutate()}
                    type="button"
                  >
                    Criar
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="Nome do grupo"
                  value={createGroupForm.name}
                  onChange={(e) =>
                    setCreateGroupForm((f) => ({ ...f, name: e.target.value }))
                  }
                />
                <textarea
                  className="input min-h-24 py-3"
                  placeholder={
                    "Participantes (DDI+numero, separados por virgula):\n5511999990001, 5511999990002"
                  }
                  value={createGroupForm.participants}
                  onChange={(e) =>
                    setCreateGroupForm((f) => ({
                      ...f,
                      participants: e.target.value,
                    }))
                  }
                />
              </FormPanel>

              <FormPanel
                title="Gerenciar Participantes"
                icon={<UserPlus className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={
                      updateGroupParticipants.isPending ||
                      !groupParticipantsForm.jid ||
                      !groupParticipantsForm.participants
                    }
                    onClick={() => updateGroupParticipants.mutate()}
                    type="button"
                  >
                    Aplicar
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="JID do grupo (ex: 12345@g.us)"
                  value={groupParticipantsForm.jid}
                  onChange={(e) =>
                    setGroupParticipantsForm((f) => ({
                      ...f,
                      jid: e.target.value,
                    }))
                  }
                />
                <select
                  className="input"
                  value={groupParticipantsForm.action}
                  onChange={(e) =>
                    setGroupParticipantsForm((f) => ({
                      ...f,
                      action: e.target.value as
                        | "add"
                        | "remove"
                        | "promote"
                        | "demote",
                    }))
                  }
                >
                  <option value="add">Adicionar</option>
                  <option value="remove">Remover</option>
                  <option value="promote">Promover a admin</option>
                  <option value="demote">Rebaixar admin</option>
                </select>
                <textarea
                  className="input min-h-20 py-3"
                  placeholder={
                    "Telefones (DDI+numero, separados por virgula):\n5511999990001, 5511999990002"
                  }
                  value={groupParticipantsForm.participants}
                  onChange={(e) =>
                    setGroupParticipantsForm((f) => ({
                      ...f,
                      participants: e.target.value,
                    }))
                  }
                />
              </FormPanel>

              <FormPanel
                title="Editar Informacoes do Grupo"
                icon={<Settings className="h-4 w-4" />}
                action={
                  <button
                    className="button-primary"
                    disabled={updateGroupInfo.isPending || !groupInfoForm.jid}
                    onClick={() => updateGroupInfo.mutate()}
                    type="button"
                  >
                    Salvar
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="JID do grupo (ex: 12345@g.us)"
                  value={groupInfoForm.jid}
                  onChange={(e) =>
                    setGroupInfoForm((f) => ({ ...f, jid: e.target.value }))
                  }
                />
                <input
                  className="input"
                  placeholder="Novo nome (opcional)"
                  value={groupInfoForm.name}
                  onChange={(e) =>
                    setGroupInfoForm((f) => ({ ...f, name: e.target.value }))
                  }
                />
                <textarea
                  className="input min-h-20 py-3"
                  placeholder="Nova descricao (opcional)"
                  value={groupInfoForm.description}
                  onChange={(e) =>
                    setGroupInfoForm((f) => ({
                      ...f,
                      description: e.target.value,
                    }))
                  }
                />
              </FormPanel>

              <FormPanel
                title="Link de Convite"
                icon={<Link className="h-4 w-4" />}
                action={
                  <button
                    className="button-secondary"
                    disabled={fetchInviteLink.isPending || !inviteLinkJID}
                    onClick={() => fetchInviteLink.mutate()}
                    type="button"
                  >
                    Obter Link
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="JID do grupo (ex: 12345@g.us)"
                  value={inviteLinkJID}
                  onChange={(e) => setInviteLinkJID(e.target.value)}
                />
                {inviteLink ? (
                  <div className="rounded-xl border border-white/10 bg-slate-900/60 p-3 text-sm break-all">
                    <span className="text-slate-400">Link: </span>
                    <a
                      href={inviteLink}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-glow hover:underline"
                    >
                      {inviteLink}
                    </a>
                  </div>
                ) : null}
              </FormPanel>

              <FormPanel
                title="Sair do Grupo"
                icon={<Ban className="h-4 w-4" />}
                action={
                  <button
                    className="button-danger"
                    disabled={leaveGroup.isPending || !leaveGroupJID}
                    onClick={() => leaveGroup.mutate()}
                    type="button"
                  >
                    Sair
                  </button>
                }
              >
                <input
                  className="input"
                  placeholder="JID do grupo (ex: 12345@g.us)"
                  value={leaveGroupJID}
                  onChange={(e) => setLeaveGroupJID(e.target.value)}
                />
              </FormPanel>
            </>
          }
        />

        {/* ── Perfil, Privacidade e Contatos ───────────────────────────── */}
        <SettingsModule
          accountSecurity={
            <>
              <div className="grid gap-6">
                <div className="panel p-6">
                  <p className="section-kicker">Conta e confianca</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Configuracoes da conta WhatsApp
                  </h2>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                    Organizamos perfil, privacidade e bloqueios em uma camada
                    separada para reforcar governanca sem poluir o fluxo
                    operacional.
                  </p>
                </div>

                <div className="grid gap-6 lg:grid-cols-2">
                  <FormPanel
                    title="Perfil Atual"
                    icon={<Smartphone className="h-4 w-4" />}
                    action={null}
                  >
                    {profileQuery.data ? (
                      <div className="space-y-3">
                        {profileQuery.data.picture_url ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img
                            src={profileQuery.data.picture_url}
                            alt="Avatar"
                            className="h-16 w-16 rounded-full border border-white/20 object-cover"
                          />
                        ) : null}
                        <dl className="space-y-3 text-sm text-slate-300">
                          <div className="flex justify-between gap-4">
                            <dt>Numero</dt>
                            <dd>{profileQuery.data.phone || "-"}</dd>
                          </div>
                          <div className="flex justify-between gap-4">
                            <dt>Nome</dt>
                            <dd>{profileQuery.data.name || "-"}</dd>
                          </div>
                        </dl>
                      </div>
                    ) : (
                      <EmptyState label="Conecte-se ao WhatsApp para ver o perfil." />
                    )}
                  </FormPanel>

                  <FormPanel
                    title="Atualizar Descricao"
                    icon={<Settings className="h-4 w-4" />}
                    action={
                      <button
                        className="button-primary"
                        disabled={updateProfile.isPending}
                        onClick={() => updateProfile.mutate()}
                        type="button"
                      >
                        Salvar
                      </button>
                    }
                  >
                    <textarea
                      className="input min-h-24 py-3"
                      placeholder="Nova descricao / status do perfil"
                      value={profileDescForm}
                      onChange={(e) => setProfileDescForm(e.target.value)}
                    />
                  </FormPanel>
                </div>

                <FormPanel
                  title="Configuracoes de Privacidade"
                  icon={<ShieldCheck className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={updatePrivacy.isPending}
                      onClick={() => updatePrivacy.mutate()}
                      type="button"
                    >
                      Salvar
                    </button>
                  }
                >
                  {privacyQuery.isLoading ? (
                    <EmptyState label="Carregando configuracoes..." />
                  ) : (
                    <div className="grid gap-4 sm:grid-cols-2">
                      {(
                        [
                          "last_seen",
                          "profile_photo",
                          "status",
                          "group_add",
                        ] as const
                      ).map((key) => (
                        <div key={key} className="flex flex-col gap-1">
                          <label className="text-xs uppercase tracking-wider text-slate-400">
                            {
                              (
                                {
                                  last_seen: "Visto por ultimo",
                                  profile_photo: "Foto de perfil",
                                  status: "Recado",
                                  group_add: "Adicionar em grupos",
                                } as Record<string, string>
                              )[key]
                            }
                          </label>
                          <select
                            className="input"
                            value={privacyForm[key]}
                            onChange={(e) =>
                              setPrivacyForm((f) => ({
                                ...f,
                                [key]: e.target.value,
                              }))
                            }
                          >
                            <option value="all">Todos</option>
                            <option value="contacts">Contatos</option>
                            <option value="none">Ninguem</option>
                          </select>
                        </div>
                      ))}
                      <div className="flex flex-col gap-1">
                        <label className="text-xs uppercase tracking-wider text-slate-400">
                          Confirmacao de Leitura
                        </label>
                        <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer mt-2">
                          <input
                            type="checkbox"
                            checked={privacyForm.read_receipts}
                            onChange={(e) =>
                              setPrivacyForm((f) => ({
                                ...f,
                                read_receipts: e.target.checked,
                              }))
                            }
                            className="rounded"
                          />
                          Ativo (mostrar duplo check azul)
                        </label>
                      </div>
                    </div>
                  )}
                </FormPanel>
              </div>

              {/* ── Bloquear / Desbloquear Contatos ─────────────────────────── */}
              <div className="grid gap-6">
                <div className="panel p-6">
                  <p className="section-kicker">Seguranca operacional</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Bloquear e desbloquear numeros
                  </h2>
                </div>

                <FormPanel
                  title="Bloquear / Desbloquear"
                  icon={<Ban className="h-4 w-4" />}
                  action={
                    <div className="flex gap-2">
                      <button
                        className="button-danger"
                        disabled={blockContact.isPending || !blockPhone}
                        onClick={() => blockContact.mutate("block")}
                        type="button"
                      >
                        Bloquear
                      </button>
                      <button
                        className="button-secondary"
                        disabled={blockContact.isPending || !blockPhone}
                        onClick={() => blockContact.mutate("unblock")}
                        type="button"
                      >
                        Desbloquear
                      </button>
                    </div>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={blockPhone}
                    onChange={(e) => setBlockPhone(e.target.value)}
                  />
                  <p className="text-xs text-slate-500">
                    O numero continuara aparecendo no historico local, mas nao
                    podera enviar mensagens.
                  </p>
                </FormPanel>
              </div>
            </>
          }
          workspaceTools={
            <>
              <div className="grid gap-6">
                <div className="panel p-6">
                  <p className="section-kicker">Ferramentas de produtividade</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Arquivar, silenciar, fixar, editar e encaminhar
                  </h2>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                    Ajustes taticos e manutencao de chat ficam aqui, separados
                    das etapas que mais ajudam a vender o produto no primeiro
                    contato.
                  </p>
                </div>

                {/* Chat Actions */}
                <FormPanel
                  title="Acoes de Chat"
                  icon={<MessageSquare className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={chatAction.isPending || !chatForm.phone}
                      onClick={() => chatAction.mutate()}
                      type="button"
                    >
                      Aplicar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={chatForm.phone}
                    onChange={(e) =>
                      setChatForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <select
                    className="input"
                    value={chatForm.action}
                    onChange={(e) =>
                      setChatForm((f) => ({
                        ...f,
                        action: e.target.value as typeof chatForm.action,
                      }))
                    }
                  >
                    <option value="archive">Arquivar chat</option>
                    <option value="unarchive">Desarquivar chat</option>
                    <option value="mute">Silenciar chat</option>
                    <option value="unmute">Dessilenciar chat</option>
                    <option value="pin">Fixar chat</option>
                    <option value="unpin">Desfixar chat</option>
                    <option value="read">Marcar como lido</option>
                    <option value="unread">Marcar como nao lido</option>
                  </select>
                  {chatForm.action === "mute" && (
                    <div className="flex gap-2 items-center">
                      <label className="text-xs text-slate-400 whitespace-nowrap">
                        Duracao (horas, 0 = sempre):
                      </label>
                      <input
                        className="input w-24"
                        type="number"
                        min="0"
                        value={chatForm.mute_hours}
                        onChange={(e) =>
                          setChatForm((f) => ({
                            ...f,
                            mute_hours: e.target.value,
                          }))
                        }
                      />
                    </div>
                  )}
                </FormPanel>

                {/* Edit Message */}
                <FormPanel
                  title="Editar Mensagem"
                  icon={<Edit2 className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        editMessage.isPending ||
                        !editForm.phone ||
                        !editForm.message_id ||
                        !editForm.new_message
                      }
                      onClick={() => editMessage.mutate()}
                      type="button"
                    >
                      Editar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={editForm.phone}
                    onChange={(e) =>
                      setEditForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="ID da mensagem (message_id)"
                    value={editForm.message_id}
                    onChange={(e) =>
                      setEditForm((f) => ({ ...f, message_id: e.target.value }))
                    }
                  />
                  <textarea
                    className="input min-h-20 py-3"
                    placeholder="Novo texto da mensagem"
                    value={editForm.new_message}
                    onChange={(e) =>
                      setEditForm((f) => ({
                        ...f,
                        new_message: e.target.value,
                      }))
                    }
                  />
                </FormPanel>
              </div>

              <div className="grid gap-6">
                <div className="panel p-6">
                  <p className="section-kicker">Acoes especiais</p>
                  <h2 className="mt-2 text-xl font-semibold text-white">
                    Operacoes avancadas
                  </h2>
                </div>

                {/* Forward Message */}
                <FormPanel
                  title="Encaminhar Mensagem"
                  icon={<Forward className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        forwardMessage.isPending ||
                        !forwardForm.phone ||
                        !forwardForm.message
                      }
                      onClick={() => forwardMessage.mutate()}
                      type="button"
                    >
                      Encaminhar
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone destino (DDI+numero)"
                    value={forwardForm.phone}
                    onChange={(e) =>
                      setForwardForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <textarea
                    className="input min-h-20 py-3"
                    placeholder="Texto a encaminhar"
                    value={forwardForm.message}
                    onChange={(e) =>
                      setForwardForm((f) => ({ ...f, message: e.target.value }))
                    }
                  />
                </FormPanel>

                {/* Star Message */}
                <FormPanel
                  title="Marcar Mensagem com Estrela"
                  icon={<Star className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={
                        starMessage.isPending ||
                        !starForm.phone ||
                        !starForm.message_id
                      }
                      onClick={() => starMessage.mutate()}
                      type="button"
                    >
                      {starForm.starred ? "Marcar ★" : "Desmarcar ★"}
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Telefone (DDI+numero)"
                    value={starForm.phone}
                    onChange={(e) =>
                      setStarForm((f) => ({ ...f, phone: e.target.value }))
                    }
                  />
                  <input
                    className="input"
                    placeholder="ID da mensagem"
                    value={starForm.message_id}
                    onChange={(e) =>
                      setStarForm((f) => ({ ...f, message_id: e.target.value }))
                    }
                  />
                  <div className="flex gap-4">
                    <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={starForm.starred}
                        onChange={(e) =>
                          setStarForm((f) => ({
                            ...f,
                            starred: e.target.checked,
                          }))
                        }
                      />
                      Marcar com estrela
                    </label>
                    <label className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={starForm.from_me}
                        onChange={(e) =>
                          setStarForm((f) => ({
                            ...f,
                            from_me: e.target.checked,
                          }))
                        }
                      />
                      Mensagem enviada por mim
                    </label>
                  </div>
                </FormPanel>

                {/* Pair Phone / Restart */}
                <FormPanel
                  title="Parear por Codigo de Telefone"
                  icon={<Smartphone className="h-4 w-4" />}
                  action={
                    <button
                      className="button-primary"
                      disabled={pairPhone.isPending || !pairPhoneForm.phone}
                      onClick={() => pairPhone.mutate()}
                      type="button"
                    >
                      Gerar Codigo
                    </button>
                  }
                >
                  <input
                    className="input"
                    placeholder="Seu numero WhatsApp (ex: 5511999999999)"
                    value={pairPhoneForm.phone}
                    onChange={(e) =>
                      setPairPhoneForm({ phone: e.target.value })
                    }
                  />
                  {pairCode && (
                    <div className="rounded-xl border border-emerald-500/40 bg-emerald-900/20 p-4">
                      <p className="text-xs text-slate-400 mb-1">
                        Codigo de pareamento (valido por 160s):
                      </p>
                      <p className="text-2xl font-mono font-bold tracking-[0.3em] text-emerald-400">
                        {pairCode}
                      </p>
                      <p className="text-xs text-slate-500 mt-2">
                        Abra WhatsApp &gt; Dispositivos conectados &gt; Conectar
                        dispositivo &gt; Conectar via numero de telefone
                      </p>
                    </div>
                  )}
                  <button
                    className="button-secondary w-full mt-2"
                    disabled={restartInstance.isPending}
                    onClick={() => restartInstance.mutate()}
                    type="button"
                  >
                    Reiniciar Instancia
                  </button>
                </FormPanel>
              </div>
            </>
          }
        />
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
  queryClient.invalidateQueries({
    queryKey: QUERY_KEYS.queueDeadLetters(tenantID),
  });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.usage(tenantID) });
  queryClient.invalidateQueries({ queryKey: QUERY_KEYS.members(tenantID) });
  if (instanceID) {
    queryClient.invalidateQueries({
      queryKey: QUERY_KEYS.audit(tenantID, instanceID),
    });
  }
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
