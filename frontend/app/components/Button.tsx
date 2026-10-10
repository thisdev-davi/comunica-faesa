import Link from "next/link";
import type { AnchorHTMLAttributes, ButtonHTMLAttributes, ReactNode } from "react";
import { Icon, type IconName } from "./Icon";
import styles from "./Button.module.css";

type Look = {
  variant?: "primary" | "accent" | "secondary" | "ghost";
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
// que respondem com redirect e para sites de fora (o Next não pode tratá-los como navegação dentro do app).
export function ButtonLink({
  href,
  reload,
  variant,
  size,
  icon,
  className,
  children,
  ...rest
}: Look & { href: string; reload?: boolean } & Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href" | "className">) {
  const content = <Content icon={icon}>{children}</Content>;
  const cls = classes({ variant, size, className });
  return reload ? (
    <a href={href} {...rest} className={cls}>
      {content}
    </a>
  ) : (
    <Link href={href} {...rest} className={cls}>
      {content}
    </Link>
  );
}
