import { redirect, unstable_rethrow } from "next/navigation";
import Link from "next/link";
import { Suspense } from "react";
import { Avatar } from "../../components/Avatar";
import { EventCard } from "../../components/EventCard";
import { JoinButton } from "../../components/JoinButton";
import { StatePanel } from "../../components/StatePanel";
import type { Attendee, CommunityEvent } from "../../events";
import { getMe } from "../../me";
import { EventPanel } from "./event-panel";
import { getEvent } from "./get-event";
import styles from "../../ideias/[id]/ideia.module.css";

// O dia (AAAA-MM-DD) no horário de Brasília: o evento sai da agenda e para de receber presença quando o dia dele acaba,
// a mesma regra da API (docs/contratos/fatia-14-presenca-em-eventos.md).
const day = (d: Date) => new Intl.DateTimeFormat("en-CA", { timeZone: "America/Sao_Paulo" }).format(d);

// Página do evento, para todo mundo, como a agenda. Roda no servidor e chama o Go direto (getEvent).
// Logado sem curso ainda não terminou o cadastro: vai para lá antes.
async function EventDetail({ params }: { params: PageProps<"/eventos/[id]">["params"] }) {
  const me = await getMe();
  if (me && !me.course) redirect("/cadastro");
  const { id } = await params;
  let event: CommunityEvent;
  try {
    event = await getEvent(id);
  } catch (err) {
    unstable_rethrow(err); // deixa passar o notFound e os erros internos do Next
    console.error("buscar evento:", err);
    return (
      <StatePanel tone="error" title="Não foi possível carregar o evento">
        Pode ser uma instabilidade rápida. Atualize a página em alguns instantes.
      </StatePanel>
    );
  }
  const past = day(new Date(event.starts_at)) < day(new Date());
  const mine = me?.id === event.author.id;
  return (
    <>
      <EventCard
        event={event}
        full
        // Visitante confirma depois de entrar; evento passado não recebe presença (a API também recusa, 409).
        action={
          me &&
          !past && <JoinButton url={`/api/events/${event.id}/attendance`} on={event.going} labels={["Vou participar", "Presença confirmada"]} />
        }
      />
      {past && <p className={styles.hint}>Este evento já aconteceu.</p>}
      {!me && !past && (
        <p className={styles.hint}>
          <Link href="/login">Entre com o Discord</Link> para confirmar presença e ver quem vai.
        </p>
      )}
      {/* O criador e o moderador editam e apagam (fatia 16); a API confere de novo. */}
      {(mine || me?.is_moderator) && <EventPanel eventId={event.id} mine={mine} />}
      {event.attendees && <Attendees people={event.attendees} past={past} />}
    </>
  );
}

// Só quem está logado recebe a lista da API: quem confirmou, em ordem de chegada.
function Attendees({ people, past }: { people: Attendee[]; past: boolean }) {
  return (
    <section className={`card ${styles.panel}`} aria-labelledby="quem-vai">
      <h2 id="quem-vai" className={styles.section}>
        Quem vai ({people.length})
      </h2>
      {people.length === 0 ? (
        <p className={styles.empty}>
          {past ? "Ninguém confirmou presença." : "Ninguém confirmou ainda. Quando alguém clicar em “Vou participar”, aparece aqui."}
        </p>
      ) : (
        <ul className={styles.list}>
          {people.map((p) => (
            <li key={p.id} className={styles.person}>
              <Link href={`/alunos/${p.id}`} className={styles.profile}>
                <Avatar name={p.name} src={p.avatar_url} decorative />
                <span className={styles.name}>{p.name}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

// O id da URL só existe na requisição: o que depende dele espera dentro do Suspense, e o resto sai no shell estático.
export default function EventPage({ params }: PageProps<"/eventos/[id]">) {
  return (
    <main id="conteudo" className={styles.page}>
      <Link href="/eventos" className={styles.back}>
        Voltar aos eventos
      </Link>
      <Suspense
        fallback={
          <p className={styles.loading} role="status">
            Carregando evento…
          </p>
        }
      >
        <EventDetail params={params} />
      </Suspense>
    </main>
  );
}
