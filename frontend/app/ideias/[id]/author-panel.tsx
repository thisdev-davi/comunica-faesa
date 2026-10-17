"use client";

import { Button, ButtonLink } from "../../components/Button";
import { useDelete } from "../../components/use-delete";
import styles from "./ideia.module.css";
import { StatusControl } from "./status-control";

const QUESTION = "Apagar esta ideia? Os interessados saem da lista e não dá para desfazer.";

// Painel "Sua ideia", só para o autor: status (fatia 7), editar e apagar.
export function AuthorPanel({ ideaId, status }: { ideaId: number; status: string }) {
  const { remove, pending, failed } = useDelete(`/api/ideas/${ideaId}`, QUESTION, "/");
  return (
    <section className={`card ${styles.panel}`} aria-labelledby="sua-ideia">
      <h2 id="sua-ideia" className={styles.section}>
        Sua ideia
      </h2>
      <StatusControl ideaId={ideaId} status={status} />
      <div className={styles.actions}>
        <ButtonLink href={`/ideias/${ideaId}/editar`} variant="secondary">
          Editar ideia
        </ButtonLink>
        <Button variant="danger" disabled={pending} onClick={remove}>
          {pending ? "Apagando…" : "Apagar ideia"}
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

// Painel "Moderação": o moderador que não é o autor só apaga (docs/contratos/fatia-15-cargo-de-moderacao.md).
export function ModeratorPanel({ ideaId }: { ideaId: number }) {
  const { remove, pending, failed } = useDelete(`/api/ideas/${ideaId}`, QUESTION, "/");
  return (
    <section className={`card ${styles.panel}`} aria-labelledby="moderacao">
      <h2 id="moderacao" className={styles.section}>
        Moderação
      </h2>
      <p className={styles.hint}>Você pode apagar esta ideia porque tem o cargo de moderação no Discord.</p>
      <div className={styles.actions}>
        <Button variant="danger" disabled={pending} onClick={remove}>
          {pending ? "Apagando…" : "Apagar ideia"}
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
