/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  output: 'standalone',
  env: {
    ORCHESTRATOR_URL: process.env.ORCHESTRATOR_URL || 'http://localhost:8080',
    GCS_BUCKET: process.env.GCS_BUCKET || 'explainiq-pdfs',
    GCS_PROJECT_ID: process.env.GCS_PROJECT_ID || '',
  },
}

module.exports = nextConfig
