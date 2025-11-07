/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  output: 'standalone',
  env: {
    // Note: ORCHESTRATOR_URL is NOT included here because it's server-side only
    // Server-side API routes read from process.env directly at runtime
    GCS_BUCKET: process.env.GCS_BUCKET || 'explainiq-pdfs',
    GCS_PROJECT_ID: process.env.GCS_PROJECT_ID || '',
  },
}

module.exports = nextConfig
