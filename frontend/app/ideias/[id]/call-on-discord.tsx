"use client";

import { useState } from "react";
import { ButtonLink } from "../../components/Button";
import styles from "./ideia.module.css";

// Abre o perfil da pessoa no Discord numa nova aba e, no mesmo clique, copia a primeira mensagem para colar na conversa.
// O Discord não tem link que abra uma conversa já escrita; o perfil (discord.com/users/<id>) é o mais perto disso.
// Se a cópia falhar (navegador sem permissão), o perfil abre do mesmo jeito.
export function CallOnDiscord({ discordId, name, ideaTitle }: { discordId: string; name: string; ideaTitle: string }) {
  const [copied, setCopied] = useState(false);
  const firstName = name.trim().split(/\s+/)[0];
  const message = `Oi, ${firstName}! Vi que você quer participar da ideia “${ideaTitle}” no Comunica FAESA. Bora conversar?`;

  return (
    <div className={styles.call}>
      <ButtonLink
        href={`https://discord.com/users/${encodeURIComponent(discordId)}`}
        reload
        target="_blank"
        rel="noopener noreferrer"
        variant="secondary"
        size="sm"
        icon="discord"
        onClick={() => navigator.clipboard?.writeText(message).then(() => setCopied(true), () => {})}
      >
        Chamar no Discord<span className="sr-only">: {name} (abre em nova aba)</span>
      </ButtonLink>
      <p className={styles.copied} role="status">
        {copied ? "Mensagem copiada, cole na conversa." : ""}
      </p>
    </div>
  );
}
