"use client";

import { useSyncExternalStore } from "react";
import { Button } from "./components/Button";
import styles from "./header.module.css";

const systemDark = () => matchMedia("(prefers-color-scheme: dark)");

// O tema mora no DOM: data-theme no <html> (escolha salva, aplicada pelo script do layout) ou, sem ela, o do sistema.
function isDark() {
  const chosen = document.documentElement.dataset.theme;
  return chosen ? chosen === "dark" : systemDark().matches;
}

// Avisa o React quando o tema muda: troca no sistema ou data-theme mexido pelo clique.
function subscribe(onChange: () => void) {
  const media = systemDark();
  media.addEventListener("change", onChange);
  const observer = new MutationObserver(onChange);
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  return () => {
    media.removeEventListener("change", onChange);
    observer.disconnect();
  };
}

export function ThemeToggle() {
  // No servidor não dá para saber o tema (null): o botão sai sem ícone e ganha o certo ao hidratar, sem erro de hidratação.
  const dark = useSyncExternalStore(subscribe, isDark, () => null);

  function toggle() {
    const next = dark ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    try {
      localStorage.setItem("tema", next);
    } catch {} // navegação privada ou armazenamento bloqueado: o tema troca, só não fica salvo
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      className={styles.iconOnly}
      icon={dark === null ? undefined : dark ? "moon" : "sun"}
      aria-label="Tema escuro"
      aria-pressed={dark ?? undefined}
      onClick={toggle}
    >
      {null}
    </Button>
  );
}
