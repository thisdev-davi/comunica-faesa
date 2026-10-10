"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { Button } from "./Button";
import styles from "./InterestButton.module.css";

// "Quero participar" alternável (docs/contratos/fatia-5-quero-participar.md). Sem aprovação: o clique já põe na lista.
// Quem diz o estado é o servidor: depois de gravar, o router.refresh() redesenha card e página com o contador certo.
// O refresh roda dentro da transição, então o botão fica desabilitado até o estado novo chegar (sem piscar o antigo).
export function InterestButton({ ideaId, interested }: { ideaId: number; interested: boolean }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [failed, setFailed] = useState(false);

  function toggle() {
    setFailed(false);
    startTransition(async () => {
      const res = await fetch(`/api/ideas/${ideaId}/interest`, { method: interested ? "DELETE" : "PUT" }).catch(() => null);
      if (res?.status === 401) {
        startTransition(() => router.push("/login")); // a sessão expirou
        return;
      }
      if (!res?.ok) {
        setFailed(true);
        return;
      }
      startTransition(() => router.refresh());
    });
  }

  return (
    <div className={styles.action}>
      <Button variant={interested ? "secondary" : "accent"} aria-pressed={interested} disabled={pending} onClick={toggle}>
        {interested ? "Interessado" : "Quero participar"}
      </Button>
      {failed && (
        <p className={styles.error} role="alert">
          Não foi possível salvar. Tente de novo.
        </p>
      )}
    </div>
  );
}
