import { Avatar } from "./components/Avatar";
import styles from "./header.module.css";
import { LoginLink } from "./login-link";
import { LogoutButton } from "./logout-button";
import { getMe } from "./me";

// Devolve os itens soltos (fragment): eles entram direto na linha flex do cabeçalho.
export async function UserMenu() {
  const me = await getMe();
  if (!me) return <LoginLink />;
  return (
    <>
      <Avatar name={me.name} src={me.avatar_url} />
      {/* aria-hidden: o avatar já dá o nome ao leitor de tela */}
      <span className={styles.name} aria-hidden="true">
        {me.name}
      </span>
      <LogoutButton />
    </>
  );
}
