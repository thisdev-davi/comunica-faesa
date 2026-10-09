import { unstable_rethrow } from "next/navigation";
import { Suspense } from "react";
import { IdeaForm } from "./idea-form";
import { API_URL, CATEGORIES, COURSES, type Idea } from "./ideas";

const dateFormat = new Intl.DateTimeFormat("pt-BR", {
  dateStyle: "short",
  timeStyle: "short",
  timeZone: "America/Sao_Paulo",
});

// Roda no servidor e chama o Go direto (sem passar pelo rewrite). Sem cache: o mural é sempre fresco.
async function IdeaList() {
  let items: Idea[];
  try {
    const res = await fetch(`${API_URL}/api/ideas`);
    if (!res.ok) throw new Error(`status ${res.status}`);
    ({ items } = await res.json());
  } catch (err) {
    unstable_rethrow(err); // deixa passar os erros internos do Next (ex.: fim da pré-renderização)
    console.error("listar ideias:", err);
    return <p role="alert">Não foi possível carregar o mural. A API está no ar?</p>;
  }

  if (items.length === 0) return <p>Nenhuma ideia postada ainda.</p>;
  return (
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
  );
}

export default function Home() {
  return (
    <main>
      <h1>Mural de ideias</h1>
      <h2>Postar ideia</h2>
      <IdeaForm />
      <h2>Ideias</h2>
      <Suspense fallback={<p>Carregando…</p>}>
        <IdeaList />
      </Suspense>
    </main>
  );
}
