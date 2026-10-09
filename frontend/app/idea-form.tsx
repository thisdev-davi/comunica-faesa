"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { CATEGORIES, COURSES, type ApiError } from "./ideas";

export function IdeaForm() {
  const router = useRouter();
  const [fields, setFields] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);

  // Sem validação no navegador de propósito: os erros exibidos são os do 400 da API.
  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = new FormData(form);
    setSending(true);
    setFields({});
    setMessage("");
    try {
      const res = await fetch("/api/ideas", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...Object.fromEntries(data), slots: Number(data.get("slots")) }),
      });
      if (res.ok) {
        form.reset();
        setMessage("Ideia postada!");
        router.refresh(); // busca o mural de novo no servidor; a ideia nova aparece no topo
        return;
      }
      const body: ApiError = await res.json();
      setFields(body.fields ?? {});
      setMessage(body.fields ? "Corrija os campos indicados." : `Erro da API: ${body.error}`);
    } catch {
      setMessage("Não foi possível falar com a API.");
    } finally {
      setSending(false);
    }
  }

  // Liga cada campo ao seu label e à sua mensagem de erro (leitores de tela anunciam os dois).
  const fieldProps = (name: string) => ({
    id: name,
    name,
    "aria-invalid": Boolean(fields[name]),
    "aria-describedby": fields[name] ? `${name}-error` : undefined,
  });
  const fieldError = (name: string) => fields[name] && <p id={`${name}-error`}>{fields[name]}</p>;

  return (
    <form onSubmit={handleSubmit}>
      <label htmlFor="title">Título</label>
      <input type="text" {...fieldProps("title")} />
      {fieldError("title")}

      <label htmlFor="description">Descrição</label>
      <textarea rows={4} {...fieldProps("description")} />
      {fieldError("description")}

      <label htmlFor="course">Curso</label>
      <select defaultValue="" {...fieldProps("course")}>
        <option value="">Selecione…</option>
        {Object.entries(COURSES).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
      {fieldError("course")}

      <label htmlFor="category">Categoria</label>
      <select defaultValue="" {...fieldProps("category")}>
        <option value="">Selecione…</option>
        {Object.entries(CATEGORIES).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
      {fieldError("category")}

      <label htmlFor="slots">Vagas</label>
      <input type="number" {...fieldProps("slots")} />
      {fieldError("slots")}

      <button type="submit" disabled={sending}>
        {sending ? "Enviando…" : "Postar ideia"}
      </button>
      <p role="status">{message}</p>
    </form>
  );
}
