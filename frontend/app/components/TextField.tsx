import { useId, type ComponentProps } from "react";
import styles from "./Field.module.css";

// Campo de texto com rótulo, dica e erro, como o Select. `rows` troca o <input> por <textarea>,
// que leva só o `name`: o resto das props (type, inputMode…) é de <input>.
export function TextField({
  label,
  hint,
  error,
  rows,
  name,
  ...rest
}: { label: string; hint?: string; error?: string; rows?: number } & ComponentProps<"input">) {
  const id = useId();
  const text = error || hint;
  const control = {
    id,
    name,
    className: styles.control,
    "aria-invalid": error ? true : undefined,
    "aria-describedby": text ? `${id}-hint` : undefined,
  };
  return (
    <div className={error ? `${styles.field} ${styles.invalid}` : styles.field}>
      <label htmlFor={id} className={styles.label}>
        {label}
      </label>
      {rows ? <textarea rows={rows} {...control} /> : <input {...rest} {...control} />}
      {text && (
        <p id={`${id}-hint`} className={styles.hint}>
          {text}
        </p>
      )}
    </div>
  );
}
