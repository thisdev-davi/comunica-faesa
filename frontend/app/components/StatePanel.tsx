import type { ReactNode } from "react";
import { Icon } from "./Icon";
import styles from "./StatePanel.module.css";

// Ocupa o lugar de uma área inteira vazia ou que falhou. O erro é anunciado na hora (alert).
export function StatePanel({ tone = "empty", title, children }: { tone?: "empty" | "error"; title: string; children: ReactNode }) {
  return (
    <div className={`card ${styles.panel} ${styles[tone]}`} role={tone === "error" ? "alert" : undefined}>
      <span className={styles.icon} aria-hidden="true">
        {tone === "error" ? <Icon name="circle-alert" size={24} /> : <span className={styles.mark} />}
      </span>
      <h2 className={styles.title}>{title}</h2>
      <p className={styles.text}>{children}</p>
    </div>
  );
}
