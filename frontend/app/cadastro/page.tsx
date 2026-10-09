import { redirect } from "next/navigation";
import { Suspense } from "react";
import { getMe } from "../me";
import { CourseForm } from "./course-form";

// Contrato da fatia 3 (docs/contratos/fatia-3-cadastro.md): só quem está logado e ainda sem curso fica aqui.
async function Signup() {
  const me = await getMe();
  if (!me) redirect("/login");
  if (me.course) redirect("/");
  return <CourseForm />;
}

export default function SignupPage() {
  return (
    <main>
      <h1>Cadastro</h1>
      <p>Antes de entrar no mural, diga qual é o seu curso.</p>
      <Suspense fallback={<p>Carregando…</p>}>
        <Signup />
      </Suspense>
    </main>
  );
}
