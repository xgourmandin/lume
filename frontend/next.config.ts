import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactCompiler: true,
  // Produce a self-contained bundle under .next/standalone so the Docker
  // image only ships the files needed to run the server (no node_modules copy).
  output: "standalone",
  turbopack: {
    // Explicitly set the workspace root to the frontend directory so Next.js
    // doesn't get confused when multiple lockfiles exist in the monorepo.
    root: __dirname,
  },
};

export default nextConfig;
