import Image from "next/image";
import styles from "./Avatar.module.css";

// Duas letras: a primeira do primeiro nome e a do último ("Ana Ribeiro" → "AR").
function initials(name: string) {
  const parts = name.trim().split(/\s+/);
  const last = parts.length > 1 ? parts[parts.length - 1][0] : "";
  return ((parts[0][0] ?? "?") + last).toUpperCase(); // nome vazio vira "?"
}

// Só o tamanho "sm" (28px): o único que o cabeçalho e o card usam. A foto vem do CDN do Discord.
export function Avatar({ name, src }: { name: string; src?: string | null }) {
  return (
    <span className={styles.avatar}>
      {src ? (
        <Image src={src} alt={name} width={28} height={28} />
      ) : (
        <>
          <span aria-hidden="true">{initials(name)}</span>
          <span className="sr-only">{name}</span>
        </>
      )}
    </span>
  );
}
