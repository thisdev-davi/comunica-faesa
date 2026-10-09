import { cookies } from "next/headers";
import { unstable_rethrow } from "next/navigation";
import { cache } from "react";
import { API_URL } from "./ideas";

// Contrato da fatia 2 (docs/contratos/fatia-2-login.md).
export type Me = {
  id: number;
  name: string;
  avatar_url: string | null;
  course: string | null;
};

// Quem está logado, ou null para visitante. Roda no servidor e repassa o cookie do navegador para a API Go.
// cache(): cabeçalho e página chamam na mesma requisição, mas a API é consultada uma vez só.
export const getMe = cache(async (): Promise<Me | null> => {
  const cookie = (await cookies()).toString();
  try {
    const res = await fetch(`${API_URL}/api/me`, { headers: { cookie } });
    if (res.status === 401) return null;
    if (!res.ok) throw new Error(`status ${res.status}`);
    return await res.json();
  } catch (err) {
    unstable_rethrow(err);
    console.error("buscar sessão:", err);
    return null; // API fora: a página segue como visitante, e o mural mostra o próprio aviso de erro
  }
});
