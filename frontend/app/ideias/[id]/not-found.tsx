import Link from "next/link";
import { StatePanel } from "../../components/StatePanel";
import styles from "./ideia.module.css";

// Aparece quando a página chama notFound(): ideia que não existe ou id inválido.
// O status HTTP fica 200 (a página já começou o streaming), e o Next marca a resposta como noindex.
export default function IdeaNotFound() {
  return (
    <main id="conteudo" className={styles.page}>
      <Link href="/" className={styles.back}>
        Voltar ao mural
      </Link>
      <StatePanel title="Ideia não encontrada">Ela pode ter sido removida, ou o link está incompleto.</StatePanel>
    </main>
  );
}
