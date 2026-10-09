import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import "./globals.css";
import { UserMenu } from "./user-menu";

export const metadata: Metadata = {
  title: "Comunica FAESA",
  description: "Mural de ideias para alunos de TI da FAESA",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="pt-BR">
      <body>
        <header>
          <Link href="/">Comunica FAESA</Link>
          <div>
            <Suspense fallback={null}>
              <UserMenu />
            </Suspense>
          </div>
        </header>
        {children}
      </body>
    </html>
  );
}
