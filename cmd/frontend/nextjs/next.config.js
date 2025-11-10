/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  output: 'standalone',
  env: {
    // Note: ORCHESTRATOR_URL is NOT included here because it's server-side only
    // Server-side API routes read from process.env directly at runtime
    // NEXT_PUBLIC_ORCHESTRATOR_URL is available on both client and server
    // This will be replaced at build time with the actual value from build args
    NEXT_PUBLIC_ORCHESTRATOR_URL: process.env.NEXT_PUBLIC_ORCHESTRATOR_URL || process.env.ORCHESTRATOR_URL || 'http://localhost:8080',
    GCS_BUCKET: process.env.GCS_BUCKET || 'explainiq-pdfs',
    GCS_PROJECT_ID: process.env.GCS_PROJECT_ID || '',
  },
  // Ensure environment variables are available at build time
  webpack: (config, { isServer }) => {
    if (!isServer) {
      // Make sure NEXT_PUBLIC_* variables are available in client bundle
      config.resolve.fallback = {
        ...config.resolve.fallback,
        fs: false,
      };
    }
    return config;
  },
}

module.exports = nextConfig
