import { cookies } from "next/headers";
import { redirect, unstable_rethrow } from "next/navigation";
import Link from "next/link";
import type { ReactNode } from "react";
import { IdeaCard } from "./components/IdeaCard";
import { StatePanel } from "./components/StatePanel";
import { API_URL, type Idea } from "./ideas";
import { getMe } from "./me";
import styles from "./mural.module.css";

// Grade de cards de uma lista da API: o mural (/api/ideas), a vitrine (/api/ideas?status=done)
// e as minhas ideias (/api/me/ideas, só para logado: o 401 manda para o login).
// Roda no servidor e chama o Go direto (sem passar pelo rewrite). Sem cache: a lista é sempre fresca.
// Repassa o cookie do navegador: sem ele a API trata todo mundo como visitante e corta a lista em 3.
// Logado sem curso ainda não terminou o cadastro: vai para lá antes de ver a lista.
export async function IdeaList({
  path,
  errorTitle,
  emptyTitle,
  emptyText,
  invite,
}: {
  path: string;
  errorTitle: string;
  emptyTitle: string;
  emptyText: ReactNode;
  invite?: string; // complemento do "Entre com Discord" para o visitante, que só vê a prévia
}) {
  const me = await getMe();
  if (me && !me.course) redirect("/cadastro");
  const cookie = (await cookies()).toString();
  let items: Idea[];
  try {
    const res = await fetch(`${API_URL}${path}`, { headers: { cookie } });
    if (res.status === 401) redirect("/login");
    if (!res.ok) throw new Error(`status ${res.status}`);
    ({ items } = await res.json());
  } catch (err) {
    unstable_rethrow(err); // deixa passar os erros internos do Next (ex.: fim da pré-renderização)
    console.error(`listar ${path}:`, err);
    return (
      <StatePanel tone="error" title={errorTitle}>
        Pode ser uma instabilidade rápida. Atualize a página em alguns instantes.
      </StatePanel>
    );
  }

  return (
    <>
      {items.length === 0 ? (
        <StatePanel title={emptyTitle}>{emptyText}</StatePanel>
      ) : (
        <ul className={styles.grid}>
          {items.map((idea) => (
            <li key={idea.id}>
              <IdeaCard idea={idea} have={me?.skills} />
            </li>
          ))}
        </ul>
      )}
      {!me && invite && (
        <p className={styles.more}>
          <Link href="/login">Entre com Discord</Link> {invite}
        </p>
      )}
    </>
  );
}

// Cards vazios no lugar da lista enquanto a API responde; o leitor de tela ouve só o "Carregando ideias…".
export function Skeleton() {
  return (
    <ul className={styles.grid} aria-busy="true">
      <li className="sr-only" role="status">
        Carregando ideias…
      </li>
      {[0, 1, 2].map((k) => (
        <li key={k} className={`card ${styles.skeleton}`} aria-hidden="true">
          <span style={{ width: "40%" }} />
          <span style={{ width: "80%", height: 18 }} />
          <span />
          <span style={{ width: "90%" }} />
          <span style={{ width: "60%" }} />
        </li>
      ))}
    </ul>
  );
}
