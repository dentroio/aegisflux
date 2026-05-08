/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  swcMinify: true,
  env: {
    ACTIONS_API_URL: process.env.ACTIONS_API_URL || 'http://localhost:8083',
    BPF_REGISTRY_URL: process.env.BPF_REGISTRY_URL || 'http://localhost:8090',
    ORCHESTRATOR_URL: process.env.ORCHESTRATOR_URL || 'http://localhost:8084',
    DECISION_API_URL: process.env.DECISION_API_URL || 'http://localhost:8087',
    INGEST_API_URL: process.env.INGEST_API_URL || 'http://localhost:9091',
    DETECTION_PIPELINE_URL: process.env.DETECTION_PIPELINE_URL || 'http://localhost:8089',
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
      {
        source: '/api/visibility/:path*',
        destination: `${process.env.INGEST_API_URL || 'http://localhost:9091'}/v1/visibility/:path*`,
      },
      {
        source: '/api/detection/:path*',
        destination: `${process.env.DETECTION_PIPELINE_URL || 'http://localhost:8089'}/v1/detection/:path*`,
      },
      {
        source: '/api/detection-packs/:path*',
        destination: `${process.env.DETECTION_PIPELINE_URL || 'http://localhost:8089'}/v1/detection-packs/:path*`,
      },
    ]
  },
}

module.exports = nextConfig
