import { cookies } from "next/headers";
import { redirect, unstable_rethrow } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";
import { IdeaCard } from "./components/IdeaCard";
import { StatePanel } from "./components/StatePanel";
import { IdeaForm } from "./idea-form";
import { API_URL, type Idea } from "./ideas";
import { getMe } from "./me";
import styles from "./mural.module.css";

const ABOUT = "Ideias de projeto postadas por alunos de TI da FAESA.";

// Roda no servidor e chama o Go direto (sem passar pelo rewrite). Sem cache: o mural é sempre fresco.
// Repassa o cookie do navegador: sem ele a API trata todo mundo como visitante e corta o mural em 3 ideias.
async function IdeaList() {
  const cookie = (await cookies()).toString();
  let items: Idea[];
  try {
    const res = await fetch(`${API_URL}/api/ideas`, { headers: { cookie } });
    if (!res.ok) throw new Error(`status ${res.status}`);
    ({ items } = await res.json());
  } catch (err) {
    unstable_rethrow(err); // deixa passar os erros internos do Next (ex.: fim da pré-renderização)
    console.error("listar ideias:", err);
    return (
      <StatePanel tone="error" title="Não foi possível carregar o mural">
        Pode ser uma instabilidade rápida. Atualize a página em alguns instantes.
      </StatePanel>
    );
  }

  return (
    <>
      {items.length === 0 ? (
        <StatePanel title="Nenhuma ideia ainda">Que tal postar a primeira? Conte o problema e quem você procura.</StatePanel>
      ) : (
        <ul className={styles.grid}>
          {items.map((idea) => (
            <li key={idea.id}>
              <IdeaCard idea={idea} />
            </li>
          ))}
        </ul>
      )}
      {!(await getMe()) && (
        <p className={styles.more}>
          <Link href="/login">Entre com Discord</Link> para ver o mural todo.
        </p>
      )}
    </>
  );
}

// Cards vazios no lugar da lista enquanto a API responde; o leitor de tela ouve só o "Carregando ideias…".
function Skeleton() {
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

// Visitante ganha o convite para entrar; o "Entrar" do cabeçalho também leva ao login.
async function Lead() {
  if (await getMe()) return <p className={styles.lead}>{ABOUT}</p>;
  return (
    <p className={styles.lead}>
      {ABOUT} <Link href="/login">Entre com o Discord</Link> para ver o mural todo e postar a sua.
    </p>
  );
}

// Só quem está logado posta; a API também recusa (401), aqui é só para não mostrar um formulário que vai falhar.
// Logado sem curso ainda não terminou o cadastro: vai para lá antes de usar o mural.
async function PostIdea() {
  const me = await getMe();
  if (me && !me.course) redirect("/cadastro");
  return me && <IdeaForm />;
}

export default function Home() {
  return (
    <main id="conteudo" className={styles.page}>
      <div className={styles.hero}>
        <h1 className={styles.display}>Mural de ideias</h1>
        <Suspense fallback={<p className={styles.lead}>{ABOUT}</p>}>
          <Lead />
        </Suspense>
      </div>
      <Suspense fallback={null}>
        <PostIdea />
      </Suspense>
      <h2 className="sr-only">Ideias</h2>
      <Suspense fallback={<Skeleton />}>
        <IdeaList />
      </Suspense>
    </main>
  );
}
