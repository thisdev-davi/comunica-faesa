"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "./components/Button";
import { Notice } from "./components/Notice";
import styles from "./header.module.css";

export function LogoutButton() {
  const router = useRouter();
  const [failed, setFailed] = useState(false);

  async function logout() {
    setFailed(false);
    const res = await fetch("/api/auth/logout", { method: "POST" }).catch(() => null);
    if (!res?.ok) {
      setFailed(true);
      return;
    }
    router.refresh(); // renderiza de novo no servidor, agora sem sessão
  }

  return (
    <>
      <Button variant="ghost" size="sm" icon="log-out" onClick={logout}>
        Sair
      </Button>
      {/* Ocupa uma linha inteira embaixo da barra do cabeçalho (flex-basis: 100%). */}
      {failed && <Notice className={styles.alert}>Não foi possível sair. Tente de novo.</Notice>}
    </>
  );
}
