"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button } from "../components/Button";
import { Notice } from "../components/Notice";
import { Select } from "../components/Select";
import { COURSES, type ApiError } from "../ideas";
import styles from "../login/login.module.css";

export function CourseForm() {
  const router = useRouter();
  const select = useRef<HTMLSelectElement>(null);
  const [invalid, setInvalid] = useState(false);
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);

  // No envio com erro, o foco vai para o campo, que já diz o que fazer.
  useEffect(() => {
    if (invalid) select.current?.focus();
  }, [invalid]);

  // Sem validação no navegador de propósito: o erro exibido é o do 400 da API.
  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const course = new FormData(e.currentTarget).get("course");
    setSending(true);
    setInvalid(false);
    setMessage("");
    try {
      const res = await fetch("/api/me", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ course }),
      });
      if (res.ok) {
        router.push("/");
        return;
      }
      const body: ApiError = await res.json();
      setInvalid(Boolean(body.fields?.course));
      setMessage(body.fields ? "Corrija o campo indicado." : "Não foi possível salvar o curso. Tente de novo.");
    } catch {
      setMessage("Não foi possível salvar o curso. Confira sua conexão e tente de novo.");
    } finally {
      setSending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className={styles.form}>
      {message && <Notice>{message}</Notice>}
      <Select
        ref={select}
        name="course"
        label="Curso"
        placeholder="Selecione…"
        options={COURSES}
        defaultValue=""
        error={invalid ? "Escolha o curso." : undefined}
      />
      <Button type="submit" disabled={sending} className={styles.wide}>
        {sending ? "Salvando…" : "Continuar"}
      </Button>
    </form>
  );
}
