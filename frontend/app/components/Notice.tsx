import type { ReactNode } from "react";
import { Icon } from "./Icon";
import styles from "./Notice.module.css";

// Aviso em bloco no topo da área afetada. danger é anunciado na hora (alert); success, como status.
export function Notice({
  tone = "danger",
  className,
  children,
}: {
  tone?: "danger" | "success";
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={[styles.notice, styles[tone], className].filter(Boolean).join(" ")} role={tone === "danger" ? "alert" : "status"}>
      <Icon name={tone === "danger" ? "circle-alert" : "circle-check"} size={20} className={styles.icon} />
      <p className={styles.text}>{children}</p>
    </div>
  );
}
