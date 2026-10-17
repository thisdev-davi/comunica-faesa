import { cookies } from "next/headers";
import { notFound } from "next/navigation";
import type { CommunityEvent } from "../../events";
import { API_URL } from "../../ideas";

// Busca um evento no servidor, repassando o cookie do navegador: só com ele a API manda `going` e a lista de quem vai.
// Evento inexistente ou id inválido vira a página "Evento não encontrado"; outro erro sobe para quem chamou.
// Quem chama precisa do unstable_rethrow no catch, para deixar passar o notFound.
export async function getEvent(id: string): Promise<CommunityEvent> {
  const cookie = (await cookies()).toString();
  const res = await fetch(`${API_URL}/api/events/${encodeURIComponent(id)}`, { headers: { cookie } });
  if (res.status === 404) notFound(); // também para id inválido (abc, 0): a API responde 404
  if (!res.ok) throw new Error(`status ${res.status}`);
  return res.json();
}
