"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button } from "./components/Button";
import { Notice } from "./components/Notice";
import { Select } from "./components/Select";
import { TextField } from "./components/TextField";
import { CATEGORIES, COURSES, type ApiError } from "./ideas";
import styles from "./idea-form.module.css";

// O que fazer em cada campo que a API marcar no 400, no lugar do texto técnico dela.
const FIX: Record<string, string> = {
  title: "Use de 3 a 120 caracteres.",
  description: "Escreva a descrição, com até 5000 caracteres.",
  course: "Escolha o curso.",
  category: "Escolha a categoria.",
  slots: "Escolha um número de 1 a 20.",
};

export function IdeaForm() {
  const router = useRouter();
  const formRef = useRef<HTMLFormElement>(null);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState<{ tone: "danger" | "success"; text: string } | null>(null);
  const [sending, setSending] = useState(false);

  // No envio com erro, o foco vai para o primeiro campo inválido (roda depois que o aria-invalid chegou ao DOM).
  useEffect(() => {
    formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
  }, [fields]);

  // Sem validação no navegador de propósito: os erros exibidos são os do 400 da API.
  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form); // antes do setSending: campos desabilitados ficam fora do FormData
    setSending(true);
    setFields({});
    setMessage(null);
    try {
      const res = await fetch("/api/ideas", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...Object.fromEntries(data), slots: Number(data.get("slots")) }),
      });
      if (res.ok) {
        form.reset();
        setMessage({ tone: "success", text: "Ideia postada. Ela já aparece no mural." });
        router.refresh(); // busca o mural de novo no servidor; a ideia nova aparece no topo
        return;
      }
      const body: ApiError = await res.json();
      setFields(body.fields ?? {});
      setMessage({
        tone: "danger",
        text: body.fields ? "Corrija os campos indicados." : "Não foi possível postar a ideia. Tente de novo.",
      });
    } catch {
      setMessage({ tone: "danger", text: "Não foi possível postar a ideia. Confira sua conexão e tente de novo." });
    } finally {
      setSending(false);
    }
  }

  const error = (name: string) => fields[name] && (FIX[name] ?? fields[name]);

  return (
    <section className={styles.card} aria-labelledby="postar-ideia">
      <h2 id="postar-ideia" className={styles.title}>
        Postar ideia
      </h2>
      <p className={styles.lead}>Conte o problema e o que você quer construir.</p>
      {message && <Notice tone={message.tone}>{message.text}</Notice>}
      <form ref={formRef} onSubmit={handleSubmit}>
        {/* fieldset disabled trava todos os campos e o botão durante o envio */}
        <fieldset disabled={sending} className={styles.fields}>
          <TextField name="title" autoComplete="off" label="Título" hint="De 3 a 120 caracteres." error={error("title")} />
          <TextField
            name="description"
            rows={6}
            label="Descrição"
            hint="O problema, a ideia e quem você procura. Até 5000 caracteres."
            error={error("description")}
          />
          <div className={styles.row}>
            <Select name="course" label="Curso" placeholder="Selecione…" options={COURSES} defaultValue="" error={error("course")} />
            <Select
              name="category"
              label="Categoria"
              placeholder="Selecione…"
              options={CATEGORIES}
              defaultValue=""
              error={error("category")}
            />
          </div>
          <TextField
            name="slots"
            type="number"
            inputMode="numeric"
            label="Quantas pessoas você procura?"
            hint="De 1 a 20."
            error={error("slots")}
          />
          <div className={styles.actions}>
            <Button type="submit" icon={sending ? undefined : "plus"}>
              {sending ? "Enviando…" : "Postar ideia"}
            </Button>
          </div>
        </fieldset>
      </form>
    </section>
  );
}
