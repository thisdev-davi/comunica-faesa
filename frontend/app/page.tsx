import { cookies } from "next/headers";
import { redirect, unstable_rethrow } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";
import { IdeaForm } from "./idea-form";
import { API_URL, CATEGORIES, COURSES, type Idea } from "./ideas";
import { getMe } from "./me";

const dateFormat = new Intl.DateTimeFormat("pt-BR", {
  dateStyle: "short",
  timeStyle: "short",
  timeZone: "America/Sao_Paulo",
});

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
    return <p role="alert">Não foi possível carregar o mural. A API está no ar?</p>;
  }

  return (
    <>
      {items.length === 0 ? (
        <p>Nenhuma ideia postada ainda.</p>
      ) : (
        <ul>
          {items.map((idea) => (
            <li key={idea.id}>
              <h3>{idea.title}</h3>
              <p>
                {COURSES[idea.course] ?? idea.course} · {CATEGORIES[idea.category] ?? idea.category} ·{" "}
                {idea.slots} {idea.slots === 1 ? "vaga" : "vagas"}
              </p>
              <p>{idea.description}</p>
              <p>
                por {idea.author.name} em{" "}
                <time dateTime={idea.created_at}>{dateFormat.format(new Date(idea.created_at))}</time>
              </p>
            </li>
          ))}
        </ul>
      )}
      {!(await getMe()) && (
        <p>
          <span aria-hidden="true">🔒</span> <Link href="/login">Entre com Discord</Link> para ver o
          mural todo.
        </p>
      )}
    </>
  );
}

// Só quem está logado posta; a API também recusa (401), aqui é só para não mostrar um formulário que vai falhar.
// Logado sem curso ainda não terminou o cadastro: vai para lá antes de usar o mural.
async function PostIdea() {
  const me = await getMe();
  if (me && !me.course) redirect("/cadastro");
  if (!me) {
    return (
      <p>
        <Link href="/login">Entre com Discord</Link> para postar uma ideia.
      </p>
    );
  }
  return <IdeaForm />;
}

export default function Home() {
  return (
    <main id="conteudo" className="provisional">
      <h1>Mural de ideias</h1>
      <h2>Postar ideia</h2>
      <Suspense fallback={<p>Carregando…</p>}>
        <PostIdea />
      </Suspense>
      <h2>Ideias</h2>
      <Suspense fallback={<p>Carregando…</p>}>
        <IdeaList />
      </Suspense>
    </main>
  );
}
