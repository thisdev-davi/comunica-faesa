import { redirect } from "next/navigation";
import { Suspense } from "react";
import { Logo } from "../components/Logo";
import styles from "../login/login.module.css";
import { getMe } from "../me";
import { CourseForm } from "./course-form";

// Contrato da fatia 3 (docs/contratos/fatia-3-cadastro.md): só quem está logado e ainda sem curso fica aqui.
async function Signup() {
  const me = await getMe();
  if (!me) redirect("/login");
  if (me.course) redirect("/");
  return <CourseForm />;
}

// Mesmo cartão do login: o cadastro é o passo seguinte a ele.
export default function SignupPage() {
  return (
    <main id="conteudo" className={`card ${styles.card}`}>
      <Logo size={48} withName={false} />
      <h1 className={styles.title}>Cadastro</h1>
      <p className={styles.lead}>Antes de entrar no mural, diga qual é o seu curso.</p>
      <Suspense fallback={<p className={styles.lead}>Carregando…</p>}>
        <Signup />
      </Suspense>
    </main>
  );
}
