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
  author: { id: number; name: string };
  created_at: string;
};

export type ApiError = {
  error: string;
  fields?: Record<string, string>;
};

export const COURSES: Record<string, string> = {
  cc: "Ciência da Computação",
  eng: "Engenharia",
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
