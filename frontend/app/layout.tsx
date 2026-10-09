import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Comunica FAESA",
  description: "Mural de ideias para alunos de TI da FAESA",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  );
}
