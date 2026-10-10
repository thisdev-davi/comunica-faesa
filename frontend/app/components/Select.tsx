import { useId, type ComponentProps } from "react";
import styles from "./Field.module.css";

// Select nativo com rótulo, dica e erro. O erro substitui a dica, pinta a borda e liga aria-invalid.
// `options` é o mapa valor → rótulo, como COURSES e CATEGORIES em ideas.ts.
export function Select({
  label,
  hint,
  error,
  options,
  placeholder,
  ...rest
}: {
  label: string;
  hint?: string;
  error?: string;
  options: Record<string, string>;
  placeholder?: string;
} & ComponentProps<"select">) {
  const id = useId();
  const text = error || hint;
  return (
    <div className={error ? `${styles.field} ${styles.invalid}` : styles.field}>
      <label htmlFor={id} className={styles.label}>
        {label}
      </label>
      <select
        {...rest}
        id={id}
        className={styles.control}
        aria-invalid={error ? true : undefined}
        aria-describedby={text ? `${id}-hint` : undefined}
      >
        {placeholder !== undefined && <option value="">{placeholder}</option>}
        {Object.entries(options).map(([value, optionLabel]) => (
          <option key={value} value={value}>
            {optionLabel}
          </option>
        ))}
      </select>
      {text && (
        <p id={`${id}-hint`} className={styles.hint}>
          {text}
        </p>
      )}
    </div>
  );
}
