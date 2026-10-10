"use client";

import { usePathname } from "next/navigation";
import { ButtonLink } from "./components/Button";

// Na própria tela de login o atalho do cabeçalho some: o botão do Discord já está ali.
export function LoginLink() {
  if (usePathname() === "/login") return null;
  return (
    <ButtonLink href="/login" variant="secondary" size="sm" icon="log-in">
      Entrar
    </ButtonLink>
  );
}
