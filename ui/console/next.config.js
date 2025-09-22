/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  env: {
    ACTIONS_API_URL: process.env.ACTIONS_API_URL || 'http://localhost:8083',
    BPF_REGISTRY_URL: process.env.BPF_REGISTRY_URL || 'http://localhost:8090',
    ORCHESTRATOR_URL: process.env.ORCHESTRATOR_URL || 'http://localhost:8084',
    DECISION_API_URL: process.env.DECISION_API_URL || 'http://localhost:8087',
  },
  async rewrites() {
    return [
      {
        source: '/api/actions/:path*',
        destination: `${process.env.ACTIONS_API_URL || 'http://localhost:8083'}/:path*`,
      },
      {
        source: '/api/registry/:path*',
        destination: `${process.env.BPF_REGISTRY_URL || 'http://localhost:8090'}/:path*`,
      },
      {
        source: '/api/orchestrator/:path*',
        destination: `${process.env.ORCHESTRATOR_URL || 'http://localhost:8084'}/:path*`,
      },
      {
        source: '/api/decision/:path*',
        destination: `${process.env.DECISION_API_URL || 'http://localhost:8087'}/:path*`,
      },
    ]
  },
}

module.exports = nextConfig
