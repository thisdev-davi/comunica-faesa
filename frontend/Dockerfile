FROM node:22-alpine
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
# o app roda sem root: a pasta é do usuário node (já vem na imagem)
RUN chown node:node /app
USER node
COPY --chown=node:node package.json package-lock.json ./
RUN npm ci
COPY --chown=node:node . .
RUN npm run build
EXPOSE 3000
# next direto, sem npm na frente: assim o SIGTERM do docker stop chega no servidor
CMD ["node_modules/.bin/next", "start", "-p", "3000"]
