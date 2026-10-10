import { cookies } from "next/headers";
import { notFound, redirect, unstable_rethrow } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";
import { IdeaCard } from "../../components/IdeaCard";
import { StatePanel } from "../../components/StatePanel";
import { API_URL, type Idea } from "../../ideas";
import { getMe } from "../../me";
import styles from "./ideia.module.css";

// Contrato da fatia 4 (docs/contratos/fatia-4-pagina-da-ideia.md): só para quem está logado e já tem curso.
// Roda no servidor e chama o Go direto, repassando o cookie do navegador (sem ele a API responde 401).
async function IdeaDetail({ params }: { params: PageProps<"/ideias/[id]">["params"] }) {
  const me = await getMe();
  if (!me) redirect("/login");
  if (!me.course) redirect("/cadastro");

  const { id } = await params;
  const cookie = (await cookies()).toString();
  let idea: Idea;
  try {
    const res = await fetch(`${API_URL}/api/ideas/${encodeURIComponent(id)}`, { headers: { cookie } });
    if (res.status === 404) notFound(); // também para id inválido (abc, 0): a API responde 404
    if (res.status === 401) redirect("/login"); // a sessão expirou entre o getMe e aqui
    if (!res.ok) throw new Error(`status ${res.status}`);
    idea = await res.json();
  } catch (err) {
    unstable_rethrow(err); // deixa passar o notFound, o redirect e os erros internos do Next
    console.error("buscar ideia:", err);
    return (
      <StatePanel tone="error" title="Não foi possível carregar a ideia">
        Pode ser uma instabilidade rápida. Atualize a página em alguns instantes.
      </StatePanel>
    );
  }
  return <IdeaCard idea={idea} full />;
}

// O id da URL só existe na requisição: o que depende dele espera dentro do Suspense, e o resto sai no shell estático.
export default function IdeaPage({ params }: PageProps<"/ideias/[id]">) {
  return (
    <main id="conteudo" className={styles.page}>
      <Link href="/" className={styles.back}>
        Voltar ao mural
      </Link>
      <Suspense
        fallback={
          <p className={styles.loading} role="status">
            Carregando ideia…
          </p>
        }
      >
        <IdeaDetail params={params} />
      </Suspense>
    </main>
  );
}
