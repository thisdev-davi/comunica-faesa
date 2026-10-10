import { redirect } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";
import { IdeaForm } from "../../idea-form";
import { getMe } from "../../me";
import styles from "../[id]/ideia.module.css";

// Só quem está logado e já tem curso posta; a API também recusa sem sessão (401).
// A pasta estática "nova" tem prioridade sobre [id]: /ideias/nova nunca vira uma ideia de id "nova".
async function PostIdea() {
  const me = await getMe();
  if (!me) redirect("/login");
  if (!me.course) redirect("/cadastro");
  return <IdeaForm />;
}

// Mesma coluna da página da ideia: o formulário e a ideia que ele cria têm a mesma largura.
export default function NewIdeaPage() {
  return (
    <main id="conteudo" className={styles.page}>
      <Link href="/" className={styles.back}>
        Voltar ao mural
      </Link>
      <Suspense
        fallback={
          <p className={styles.loading} role="status">
            Carregando…
          </p>
        }
      >
        <PostIdea />
      </Suspense>
    </main>
  );
}
