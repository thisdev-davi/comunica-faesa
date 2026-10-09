"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

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
      <button type="button" onClick={logout}>
        Sair
      </button>
      {failed && <span role="alert">Não foi possível sair. Tente de novo.</span>}
    </>
  );
}
