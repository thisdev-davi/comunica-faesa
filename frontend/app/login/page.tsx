import { redirect } from "next/navigation";
import { Suspense } from "react";
import { getMe } from "../me";

// Motivos que a API manda em /login?erro=... quando o login não dá certo.
const errors: Record<string, string> = {
  cancelado: "Login cancelado no Discord. Tente de novo quando quiser.",
  fora_do_servidor: "Você precisa estar no servidor da comunidade no Discord para entrar.",
  falhou: "Não foi possível entrar. Tente de novo.",
};

async function LoginBox({ searchParams }: Pick<PageProps<"/login">, "searchParams">) {
  if (await getMe()) redirect("/");

  const { erro } = await searchParams;
  const message = typeof erro === "string" && Object.hasOwn(errors, erro) ? errors[erro] : null;
  const invite = process.env.DISCORD_INVITE_URL;
  return (
    <>
      {message && (
        <p role="alert">
          {message}
          {erro === "fora_do_servidor" && invite && (
            <>
              {" "}
              <a href={invite}>Entre no servidor da comunidade</a> e depois tente de novo.
            </>
          )}
        </p>
      )}
      {/* <a> e não <Link>: a API responde com redirect para o Discord, precisa ser navegação completa. */}
      <a href="/api/auth/discord">Entrar com Discord</a>
    </>
  );
}

export default function LoginPage({ searchParams }: PageProps<"/login">) {
  return (
    <main>
      <h1>Entrar</h1>
      <p>Use sua conta do Discord, a mesma do servidor da comunidade.</p>
      <Suspense fallback={<p>Carregando…</p>}>
        <LoginBox searchParams={searchParams} />
      </Suspense>
    </main>
  );
}
