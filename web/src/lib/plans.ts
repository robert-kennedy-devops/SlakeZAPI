export type PlanOption = {
  value: "trial" | "starter" | "growth" | "pro";
  code: "Trial" | "Starter" | "Growth" | "Pro";
  name: string;
  badge?: string;
  price: string;
  monthlyLimit: string;
  webhook: string;
  summary: string;
  idealFor: string;
  details: string;
  highlights: string[];
};

export const PLAN_OPTIONS: PlanOption[] = [
  {
    value: "trial",
    code: "Trial",
    name: "Degustacao",
    badge: "Gratis por 2 dias",
    price: "R$ 0",
    monthlyLimit: "Todas as funcionalidades por 48 horas",
    webhook: "Incluido durante o teste",
    summary: "Acesso de degustacao para explorar o produto completo antes de contratar um plano pago.",
    idealFor: "Quem quer validar a experiencia, testar integracoes e apresentar a operacao para a equipe antes de assinar.",
    details: "48 horas gratis • todos os recursos desbloqueados • sem cobranca inicial",
    highlights: [
      "Teste completo por 2 dias",
      "Webhooks e automacao liberados",
      "Sem necessidade de compromisso imediato",
      "Ideal para prova rapida de valor",
    ],
  },
  {
    value: "starter",
    code: "Starter",
    name: "Essencial",
    price: "R$ 79/mês",
    monthlyLimit: "3.000 mensagens/mês",
    webhook: "Nao incluido",
    summary: "Entrada profissional para operar WhatsApp com mais organizacao e menos improviso.",
    idealFor: "Pequenas operacoes, negocios locais, consultorios e equipes iniciando estrutura comercial.",
    details: "3.000 mensagens/mês • inbox e campanhas • sem webhook",
    highlights: [
      "Inbox operacional",
      "Campanhas e envio de mídia",
      "Conexao de instancias",
      "Ideal para validar a operacao",
    ],
  },
  {
    value: "growth",
    code: "Growth",
    name: "Profissional",
    badge: "Mais vendido",
    price: "R$ 149/mês",
    monthlyLimit: "15.000 mensagens/mês",
    webhook: "Incluido",
    summary: "O melhor equilibrio entre preco, automacao e operacao para empresas em crescimento.",
    idealFor: "Empresas com atendimento ativo, campanhas recorrentes e necessidade de integrar sistemas.",
    details: "15.000 mensagens/mês • webhooks • automacao e operacao",
    highlights: [
      "Webhooks habilitados",
      "Campanhas, grupos e automacao",
      "Mais previsibilidade operacional",
      "Melhor custo-beneficio comercial",
    ],
  },
  {
    value: "pro",
    code: "Pro",
    name: "Escala",
    price: "R$ 299/mês",
    monthlyLimit: "60.000 mensagens/mês",
    webhook: "Incluido",
    summary: "Camada de escala para operacoes com maior volume, mais rotinas e mais exigencia de controle.",
    idealFor: "Operacoes de vendas, suporte e notificacao com maior volume e times mais estruturados.",
    details: "60.000 mensagens/mês • webhooks • auditoria e escala",
    highlights: [
      "Webhooks e auditoria",
      "Operacao multi-instancia",
      "Pronto para rotina intensa",
      "Maior folego para crescimento",
    ],
  },
];
