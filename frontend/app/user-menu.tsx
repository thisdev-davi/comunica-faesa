import Image from "next/image";
import Link from "next/link";
import { LogoutButton } from "./logout-button";
import { getMe } from "./me";

export async function UserMenu() {
  const me = await getMe();
  if (!me) return <Link href="/login">Entrar</Link>;
  return (
    <>
      {/* alt vazio: o nome já aparece ao lado, a foto é decorativa para leitor de tela */}
      {me.avatar_url && <Image src={me.avatar_url} alt="" width={32} height={32} />}
      <span>{me.name}</span>
      <LogoutButton />
    </>
  );
}
