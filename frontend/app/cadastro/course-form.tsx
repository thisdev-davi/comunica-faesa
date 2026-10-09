"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { COURSES, type ApiError } from "../ideas";

export function CourseForm() {
  const router = useRouter();
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);

  // Sem validação no navegador de propósito: o erro exibido é o do 400 da API.
  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const course = new FormData(e.currentTarget).get("course");
    setSending(true);
    setError("");
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
      setError(body.fields?.course ?? "");
      setMessage(body.fields ? "Corrija o campo indicado." : `Erro da API: ${body.error}`);
    } catch {
      setMessage("Não foi possível falar com a API.");
    } finally {
      setSending(false);
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <label htmlFor="course">Curso</label>
      <select
        id="course"
        name="course"
        defaultValue=""
        aria-invalid={Boolean(error)}
        aria-describedby={error ? "course-error" : undefined}
      >
        <option value="">Selecione…</option>
        {Object.entries(COURSES).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </select>
      {error && <p id="course-error">{error}</p>}

      <button type="submit" disabled={sending}>
        {sending ? "Salvando…" : "Continuar"}
      </button>
      <p role="status">{message}</p>
    </form>
  );
}
