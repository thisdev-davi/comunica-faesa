"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

// Apagar de vez (a ideia na fatia 6 e 15, o evento na 16): confirma na caixa nativa do navegador (já acessível,
// sem modal próprio), manda o DELETE e vai para `back`. 404 é "já tinha sido apagado" (em outra aba, por exemplo):
// o resultado é o mesmo, então também segue.
export function useDelete(url: string, question: string, back: string) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();
  const [failed, setFailed] = useState(false);

  function remove() {
    if (!confirm(question)) return;
    setFailed(false);
    startTransition(async () => {
      const res = await fetch(url, { method: "DELETE" }).catch(() => null);
      if (res?.status === 401) {
        startTransition(() => router.push("/login")); // a sessão expirou
        return;
      }
      if (!res?.ok && res?.status !== 404) {
        setFailed(true);
        return;
      }
      startTransition(() => router.push(back));
    });
  }

  return { remove, pending, failed };
}
