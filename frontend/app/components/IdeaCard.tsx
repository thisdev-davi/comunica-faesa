import Link from "next/link";
import { CATEGORIES, COURSES, type Idea } from "../ideas";
import { Avatar } from "./Avatar";
import styles from "./IdeaCard.module.css";

const STATUS: Record<string, { label: string; tone: string }> = {
  open: { label: "Aberta", tone: styles.open },
  in_progress: { label: "Em andamento", tone: styles.progress },
  done: { label: "Concluída", tone: styles.done },
};

// "postada em DD/MM", no horário de Brasília.
const dayMonth = new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "2-digit", timeZone: "America/Sao_Paulo" });

// StatusBadge do design system: recebe o status cru da API e sempre mostra a palavra junto da cor.
function StatusBadge({ status }: { status: string }) {
  const s = STATUS[status] ?? STATUS.open;
  return (
    <span className={`${styles.badge} ${s.tone}`}>
      <span className={styles.mark} aria-hidden="true" />
      {s.label}
    </span>
  );
}

// Só o que a API devolve hoje: sem habilidades, sem @ do Discord e sem "Quero participar".
// No mural, o título leva à página da ideia e a descrição corta em três linhas.
// `full` é a própria página da ideia: título vira h1, descrição inteira e sem link (já estamos nela).
export function IdeaCard({ idea, full = false }: { idea: Idea; full?: boolean }) {
  const Title = full ? "h1" : "h3";
  return (
    <article className={`card ${styles.card} ${full ? styles.full : ""}`}>
      <header className={styles.head}>
        <p className={styles.eyebrow}>
          {COURSES[idea.course] ?? idea.course} · {CATEGORIES[idea.category] ?? idea.category}
        </p>
        <StatusBadge status={idea.status} />
      </header>
      <Title className={styles.title}>
        {full ? (
          idea.title
        ) : (
          <Link href={`/ideias/${idea.id}`} className={styles.link}>
            {idea.title}
          </Link>
        )}
      </Title>
      <p className={full ? styles.desc : `${styles.desc} ${styles.clamp}`}>{idea.description}</p>
      <footer className={styles.foot}>
        <Avatar name={idea.author.name} />
        <div>
          <p className={styles.author}>{idea.author.name}</p>
          <p className={styles.meta}>
            Precisa de {idea.slots} {idea.slots === 1 ? "pessoa" : "pessoas"} · postada em{" "}
            <time dateTime={idea.created_at}>{dayMonth.format(new Date(idea.created_at))}</time>
          </p>
        </div>
      </footer>
    </article>
  );
}
