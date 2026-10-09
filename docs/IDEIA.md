# Plataforma de Projetos de TI — FAESA

8 de outubro de 2026 · Davi de Souza

Documento da ideia. Referência durante o desenvolvimento, organizado por fase: MVP, Próximo Passo e Futuro.

## Visão e problema

**Em uma frase:** uma ponte para o aluno de TI da FAESA achar parceiro de projeto e ter mais oportunidades de fazer coisa junto, de verdade.

Alunos de TI (Ciência da Computação, Engenharia e ADS) querem fazer mais projetos e eventos da área, mas esbarram em quatro coisas: falta gente para montar time, falta comunicação entre quem tem ideia e quem quer participar, falta um meio onde isso aconteça, e há poucas oportunidades de botar a mão na massa (o hackathon acontece cerca de uma vez por ano).

A plataforma ataca duas dores centrais:

- Achar gente para tirar ideias do papel.
- Ter mais momentos de botar a mão na massa.

## MVP — o coração

Duas pernas que se apoiam: o mural de ideias e a aba de eventos.

**Login**

- Entrada via Discord (OAuth). A conversa dos grupos migra para o Discord central da comunidade — a plataforma não constrói chat.

**Mural de ideias**

- O aluno posta uma ideia com curso, categoria, descrição, quantas pessoas precisa e habilidades procuradas.
- Os outros podem comentar.
- Botão "quero participar", sem aprovação — o espírito é conhecer gente nova, então a lista de interessados fica aberta.
- O autor vê a lista de interessados numa aba e chama a galera pelo Discord.

**Status da ideia (controlado pelo próprio autor)**

- Aberta, em andamento, concluída.
- O autor decide quando trava novos interessados.
- Concluída vai para uma vitrine de projetos concluídos.

**Perfil do aluno**

- Básico: nome, foto, curso e ideias que postou ou participou.
- Habilidades no perfil (front, back, design, dados, etc.).

**Descoberta**

- Filtros por curso, categoria, habilidade procurada e status.
- Busca por texto.
- Habilidades na ideia e no perfil dão sentido aos filtros e melhoram o encontro entre pessoas.

**Aba de eventos**

- Eventos são independentes: não precisam nascer de uma ideia.
- Podem ser dia de trabalho de um projeto, ou algo amplo da comunidade (hackathon, roda de conversa, encontro de líderes).
- Campos: título, data, local e confirmação de presença ("vou").

## Próximo passo — logo após o MVP

- Link do repositório na vitrine de concluídos — transforma a vitrine em portfólio para entrevistas.
- "Tô disponível": para o aluno que não tem ideia mas quer entrar em algum projeto (o inverso do mural).
- Tags livres na ideia (ex.: jogos, IA, mobile), além de curso e categoria.

## Futuro — junto com o professor ou vira rede social

- Aba de professor com avaliação acadêmica das ideias (alguns projetos precisam de professor, outros não).
- Bot do Discord que cria call automática quando o grupo fecha.
- Build in public: acompanhamento diário do que o time fez, gera engajamento.
- Seguir pessoas (vira feed e notificações).
- Sugestão de convidar amigo do Discord.
- Líderes de turma como embaixadores (divulgam e moderam).
- Problemas reais postados pela faculdade ou comunidade (combina com o Projeto Integrador).
- Hackathons e desafios internos com tema e prazo, usando a aba de eventos.

## Fica de fora (de propósito) e régua de decisão

**Fica de fora**

- Chat e troca de ideia livre (rodas de conversa, canais de tema) — o Discord já faz bem; a plataforma só aponta para lá.

**Régua de decisão**

A pergunta não é "isso é legal?", porque quase tudo é. A pergunta é: "sem isso, a dor principal continua sem solução?". Se a resposta é não, é desejável, não essencial — e desejável vai para a fila.
