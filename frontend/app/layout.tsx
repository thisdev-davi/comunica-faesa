import type { Metadata } from "next";
import { Plus_Jakarta_Sans } from "next/font/google";
import Link from "next/link";
import { Suspense } from "react";
import { Logo } from "./components/Logo";
import "./globals.css";
import styles from "./header.module.css";
import { ThemeToggle } from "./theme-toggle";
import { UserMenu } from "./user-menu";

// next/font baixa a fonte no build e serve do próprio site; o globals.css lê a variável em --font-sans.
const jakarta = Plus_Jakarta_Sans({ subsets: ["latin"], variable: "--font-jakarta" });

export const metadata: Metadata = {
  title: "Comunica FAESA",
  description: "Mural de ideias para alunos de TI da FAESA",
};

// Aplica o tema salvo antes da primeira pintura (sem piscar o tema do sistema). Roda enquanto o navegador lê o <head>.
const THEME_SCRIPT = `try{var t=localStorage.getItem("tema");if(t)document.documentElement.dataset.theme=t}catch(e){}`;

// Cada página põe id="conteudo" no seu <main>, o alvo do "Pular para o conteúdo".
// suppressHydrationWarning: o script acima pode pôr data-theme no <html> antes de o React hidratar.
export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="pt-BR" className={jakarta.variable} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_SCRIPT }} />
      </head>
      <body>
        <a className={styles.skip} href="#conteudo">
          Pular para o conteúdo
        </a>
        {/* A barra ocupa a tela toda; o conteúdo dela (.bar) tem a largura do mural, alinhado às bordas da grade. */}
        <header className={styles.head}>
          <div className={styles.bar}>
            <Link href="/" className={styles.brand}>
              <Logo />
            </Link>
            <ThemeToggle />
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
