import { redirect, unstable_rethrow } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";
import { StatePanel } from "../../../components/StatePanel";
import type { CommunityEvent } from "../../../events";
import { getMe } from "../../../me";
import { EventForm } from "../../event-form";
import { getEvent } from "../get-event";
import styles from "../../../ideias/[id]/ideia.module.css";

// Contrato da fatia 16 (docs/contratos/fatia-16-editar-e-apagar-evento.md): o formulário do criar, já preenchido.
// Só o criador ou um moderador; os outros voltam para a página do evento (a API também recusa, 403).
async function EditEvent({ params }: { params: PageProps<"/eventos/[id]/editar">["params"] }) {
  const me = await getMe();
  if (!me) redirect("/login");
  if (!me.course) redirect("/cadastro");

  const { id } = await params;
  let event: CommunityEvent;
  try {
    event = await getEvent(id);
  } catch (err) {
    unstable_rethrow(err); // deixa passar o notFound, o redirect e os erros internos do Next
    console.error("buscar evento para editar:", err);
    return (
      <StatePanel tone="error" title="Não foi possível carregar o evento">
        Pode ser uma instabilidade rápida. Atualize a página em alguns instantes.
      </StatePanel>
    );
  }
  if (event.author.id !== me.id && !me.is_moderator) redirect(`/eventos/${event.id}`);
  return (
    <>
      <Link href={`/eventos/${event.id}`} className={styles.back}>
        Voltar ao evento
      </Link>
      <EventForm event={event} />
    </>
  );
}

// O link de volta precisa do id, que só existe na requisição: ele sai junto do formulário, dentro do Suspense.
export default function EditEventPage({ params }: PageProps<"/eventos/[id]/editar">) {
  return (
    <main id="conteudo" className={styles.page}>
      <Suspense
        fallback={
          <p className={styles.loading} role="status">
            Carregando evento…
          </p>
        }
      >
        <EditEvent params={params} />
      </Suspense>
    </main>
  );
}
