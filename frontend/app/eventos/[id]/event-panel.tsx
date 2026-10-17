"use client";

import { Button, ButtonLink } from "../../components/Button";
import { useDelete } from "../../components/use-delete";
import styles from "../../ideias/[id]/ideia.module.css";

// Painel "Seu evento" (o criador) ou "Moderação" (moderador que não é o criador), com editar e apagar
// (docs/contratos/fatia-16-editar-e-apagar-evento.md).
export function EventPanel({ eventId, mine }: { eventId: number; mine: boolean }) {
  const { remove, pending, failed } = useDelete(
    `/api/events/${eventId}`,
    "Apagar este evento? Quem confirmou presença sai da lista e não dá para desfazer.",
    "/eventos",
  );

  return (
    <section className={`card ${styles.panel}`} aria-labelledby="painel-evento">
      <h2 id="painel-evento" className={styles.section}>
        {mine ? "Seu evento" : "Moderação"}
      </h2>
      {!mine && <p className={styles.hint}>Você pode editar e apagar este evento porque tem o cargo de moderação no Discord.</p>}
      <div className={styles.actions}>
        <ButtonLink href={`/eventos/${eventId}/editar`} variant="secondary">
          Editar evento
        </ButtonLink>
        <Button variant="danger" disabled={pending} onClick={remove}>
          {pending ? "Apagando…" : "Apagar evento"}
        </Button>
      </div>
      {failed && (
        <p className={styles.error} role="alert">
          Não foi possível apagar. Tente de novo.
        </p>
      )}
    </section>
  );
}
