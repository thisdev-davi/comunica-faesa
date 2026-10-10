import styles from "./Logo.module.css";

// Dois losangos que se cruzam: duas pessoas que se encontram, e o cruzamento é o projeto.
// As cores vêm dos tokens, então o símbolo acompanha o tema. Sem o nome, o símbolo vira imagem com rótulo.
export function Logo({ size = 28, withName = true }: { size?: number; withName?: boolean }) {
  return (
    <span className={styles.logo} translate="no">
      <svg
        viewBox="0 0 32 32"
        width={size}
        height={size}
        {...(withName ? { "aria-hidden": true } : { role: "img", "aria-label": "Comunica FAESA" })}
      >
        <rect className={styles.a} x={6} y={9.5} width={13} height={13} rx={3.5} transform="rotate(45 12.5 16)" />
        <rect className={styles.b} x={13} y={9.5} width={13} height={13} rx={3.5} transform="rotate(45 19.5 16)" />
        <rect className={styles.x} x={11.975} y={11.975} width={8.05} height={8.05} rx={1.5} transform="rotate(45 16 16)" />
      </svg>
      {withName && "Comunica FAESA"}
    </span>
  );
}
