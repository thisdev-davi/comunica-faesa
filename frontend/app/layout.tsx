import type { Metadata } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import Link from "next/link";
import { Suspense } from "react";
import "./globals.css";
import { UserMenu } from "./user-menu";

// next/font baixa a fonte no build e serve do próprio site; o globals.css lê a variável em --font-sans.
const jakarta = Plus_Jakarta_Sans({ subsets: ["latin"], variable: "--font-jakarta" });

export const metadata: Metadata = {
  title: "Comunica FAESA",
  description: "Mural de ideias para alunos de TI da FAESA",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="pt-BR" className={jakarta.variable}>
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
