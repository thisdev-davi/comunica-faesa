import type { NextConfig } from "next";
import { API_URL } from "./app/ideas";

const nextConfig: NextConfig = {
  cacheComponents: true,
  partialPrefetching: true,
  // O navegador chama /api/* no mesmo domínio do front e o Next repassa para o Go. Sem CORS.
  // Em produção, o Caddy faz o mesmo encaminhamento.
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${API_URL}/api/:path*` }];
  },
};

export default nextConfig;
