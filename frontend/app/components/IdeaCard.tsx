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
// A descrição aparece inteira: o corte em três linhas do design system espera a página da ideia, que ainda não existe.
export function IdeaCard({ idea }: { idea: Idea }) {
  return (
    <article className={`card ${styles.card}`}>
      <header className={styles.head}>
        <p className={styles.eyebrow}>
          {COURSES[idea.course] ?? idea.course} · {CATEGORIES[idea.category] ?? idea.category}
        </p>
        <StatusBadge status={idea.status} />
      </header>
      <h3 className={styles.title}>{idea.title}</h3>
      <p className={styles.desc}>{idea.description}</p>
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
