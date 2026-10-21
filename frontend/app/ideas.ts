// Tipos e listas do contrato da fatia 1 (docs/contratos/fatia-1-mural.md).

// Endereço da API Go para quem roda no servidor (next.config e Server Components).
// O navegador nunca usa isso: ele chama /api/* e o rewrite repassa.
export const API_URL = process.env.API_URL ?? "http://localhost:8090";

export type Idea = {
  id: number;
  title: string;
  description: string;
  course: string;
  category: string;
  slots: number;
  status: string;
  author: { id: number; name: string; avatar_url: string | null };
  created_at: string;
  done_at: string | null; // fatia 7 (docs/contratos/fatia-7-status.md): quando foi concluída
  skills: string[]; // fatia 12: habilidades procuradas, na ordem do catálogo
  interest_count: number; // fatia 5 (docs/contratos/fatia-5-quero-participar.md)
  interested: boolean; // quem pede está na lista
  interest_faces: Face[]; // os 8 primeiros interessados; [] para visitante
  interests?: Interest[]; // só vem para o autor
  call_url?: string; // fatia 19: canal do grupo no Discord, só para o autor e a lista, e só no GET de uma ideia
};

export type Face = {
  id: number;
  name: string;
  avatar_url: string | null;
};

export type Interest = {
  id: number;
  name: string;
  avatar_url: string | null;
  discord_id: string | null;
  skills: string[]; // fatia 11 (docs/contratos/fatia-11-habilidades.md)
};

export type ApiError = {
  error: string;
  fields?: Record<string, string>;
};

// Status são sempre "Aberta", "Em andamento" e "Concluída" (docs/DESIGN.md).
export const STATUSES: Record<string, string> = {
  open: "Aberta",
  in_progress: "Em andamento",
  done: "Concluída",
};

export const COURSES: Record<string, string> = {
  cc: "Ciência da Computação",
  eng: "Engenharia",
  ads: "ADS",
};

// Sigla do curso para a etiqueta do card, que precisa caber numa linha; a página da ideia usa o nome inteiro.
export const COURSE_SHORT: Record<string, string> = {
  cc: "CC",
  eng: "ENG",
  ads: "ADS",
};

export const CATEGORIES: Record<string, string> = {
  web: "Web",
  mobile: "Mobile",
  ai: "IA",
  data: "Dados",
  games: "Jogos",
  competitive_programming: "Maratona de Programação",
  security: "Segurança",
  iot: "IoT / Hardware",
  other: "Outro",
};

// Catálogo de habilidades da fatia 11 (docs/contratos/fatia-11-habilidades.md), na ordem da API.
export const SKILLS: Record<string, string> = {
  competitive_programming: "Maratona de Programação",
  frontend: "Front-end",
  backend: "Back-end",
  mobile: "Mobile",
  design: "Design (UI/UX)",
  data: "Dados",
  ai: "IA",
  devops: "DevOps / Infra",
  games: "Jogos",
  security: "Segurança",
  hardware: "Hardware / IoT",
  testing: "Testes / QA",
  management: "Gestão / Produto",
};
