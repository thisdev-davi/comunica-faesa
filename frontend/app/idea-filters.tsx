import { Button, ButtonLink } from "./components/Button";
import { Select } from "./components/Select";
import { CATEGORIES, COURSES, SKILLS, STATUSES } from "./ideas";
import styles from "./mural.module.css";

// Filtros do mural (docs/contratos/fatia-17-filtros-do-mural.md). As chaves são as da API: o front repassa.
// A concluída fica na vitrine, então o mural só filtra entre aberta e em andamento.
const FILTERS = {
  status: { label: "Status", all: "Todos", options: { open: STATUSES.open, in_progress: STATUSES.in_progress } },
  course: { label: "Curso", all: "Todos", options: COURSES },
  category: { label: "Categoria", all: "Todas", options: CATEGORIES },
  skill: { label: "Habilidade", all: "Todas", options: SKILLS },
};

// Só os filtros com valor da lista: valor estranho na URL some aqui, antes de virar 400 na API.
export function pickFilters(params: Record<string, string | string[] | undefined>) {
  const picked = new URLSearchParams();
  for (const [key, { options }] of Object.entries(FILTERS)) {
    const v = params[key];
    if (typeof v === "string" && Object.hasOwn(options, v)) picked.set(key, v);
  }
  return picked;
}

// Formulário GET nativo: funciona sem JS e deixa o filtro na URL. O key pela query remonta o form
// quando a URL muda por navegação do Next ("Limpar filtros"), senão os selects guardariam o valor velho.
export function IdeaFilters({ values }: { values: URLSearchParams }) {
  return (
    <form method="get" action="/" role="search" aria-label="Filtrar ideias" className={styles.filters} key={values.toString()}>
      {Object.entries(FILTERS).map(([key, { label, all, options }]) => (
        <Select key={key} name={key} label={label} options={options} placeholder={all} defaultValue={values.get(key) ?? ""} />
      ))}
      <div className={styles.filterActions}>
        <Button type="submit" variant="secondary">
          Filtrar
        </Button>
        {values.size > 0 && (
          <ButtonLink href="/" variant="ghost">
            Limpar filtros
          </ButtonLink>
        )}
      </div>
    </form>
  );
}
