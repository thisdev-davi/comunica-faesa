import Link from "next/link";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Icon, type IconName } from "./Icon";
import styles from "./Button.module.css";

// Sem "accent": o único uso dele no design system é o "Quero participar", que ainda não existe.
type Look = {
  variant?: "primary" | "secondary" | "ghost";
  size?: "md" | "sm";
  icon?: IconName;
  className?: string;
  children: ReactNode;
};

function classes({ variant = "primary", size = "md", className }: Omit<Look, "children">) {
  return [styles.btn, styles[variant], styles[size], className].filter(Boolean).join(" ");
}

// O texto vai em caixa normal; a caixa alta vem do CSS, assim o leitor de tela não soletra.
function Content({ icon, children }: Pick<Look, "icon" | "children">) {
  return (
    <>
      {icon && (
        <span className={styles.icon}>
          <Icon name={icon} />
        </span>
      )}
      {children}
    </>
  );
}

export function Button({ variant, size, icon, className, children, ...rest }: Look & ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button type="button" {...rest} className={classes({ variant, size, className })}>
      <Content icon={icon}>{children}</Content>
    </button>
  );
}

// Link com cara de botão: navega, não age. `reload` troca o <Link> por <a> para rotas da API
// que respondem com redirect (o Next não pode tratá-las como navegação dentro do app).
export function ButtonLink(props: Look & { href: string; reload?: boolean }) {
  const { href, reload, icon, children } = props;
  const content = <Content icon={icon}>{children}</Content>;
  return reload ? (
    <a href={href} className={classes(props)}>
      {content}
    </a>
  ) : (
    <Link href={href} className={classes(props)}>
      {content}
    </Link>
  );
}
