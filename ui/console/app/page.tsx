'use client'

import { useState, useEffect } from 'react'
import { 
  Shield, 
  Network, 
  Settings, 
  Activity, 
  Users, 
  AlertTriangle,
  CheckCircle,
  Clock,
  Server
} from 'lucide-react'

interface Agent {
  agent_uid: string
  org_id: string
  host_id: string
  hostname: string
  agent_version: string
  platform: {
    os: string
    kernel_version: string
    architecture: string
    primary_ip: string
  }
  network: {
    primary_ip: string
    mac_address: string
    subnet: string
  }
  labels: string[]
  created: string
  last_seen: string
  status: 'online' | 'offline' | 'unknown'
}

interface Artifact {
  id: string
  name: string
  version: string
  description: string
  type: string
  architecture: string
  created_at: string
  size: number
  metadata: {
    policy_type?: string
    target_ip?: string
    protocol?: string
    direction?: string
  }
  hosts: string[]
}

export default function Dashboard() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [artifacts, setArtifacts] = useState<Artifact[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchDashboardData()
    const interval = setInterval(fetchDashboardData, 30000) // Refresh every 30 seconds
    return () => clearInterval(interval)
  }, [])

  const fetchDashboardData = async () => {
    try {
      setLoading(true)
      setError(null)

      // Fetch agents
      const agentsResponse = await fetch('/api/actions/agents')
      if (agentsResponse.ok) {
        const agentsData = await agentsResponse.json()
        // Ensure all agents have a status property and normalize data structure
        const agentsWithStatus = (agentsData.agents || []).map((agent: any) => ({
          ...agent,
          status: agent.status || 'unknown',
          // Normalize platform structure for different agent formats
          platform: {
            ...agent.platform,
            primary_ip: agent.platform?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            hostname: agent.platform?.hostname || agent.hostname,
            os: agent.platform?.os || 'linux',
            kernel_version: agent.platform?.kernel_version || 'unknown',
            architecture: agent.platform?.architecture || agent.platform?.arch || 'unknown'
          }
        }))
        // Deduplicate agents by host_id to show unique hosts
        const uniqueAgents = agentsWithStatus.reduce((acc: any[], agent: any) => {
          const existing = acc.find(a => a.host_id === agent.host_id)
          if (!existing || new Date(agent.last_seen) > new Date(existing.last_seen)) {
            return acc.filter(a => a.host_id !== agent.host_id).concat(agent)
          }
          return acc
        }, [])
        setAgents(uniqueAgents)
      }

      // Fetch artifacts
      const artifactsResponse = await fetch('/api/registry/artifacts')
      if (artifactsResponse.ok) {
        const artifactsData = await artifactsResponse.json()
        // Ensure all artifacts have a hosts property
        const artifactsWithHosts = (artifactsData.artifacts || []).map((artifact: any) => ({
          ...artifact,
          hosts: artifact.hosts || []
        }))
        setArtifacts(artifactsWithHosts)
      }
    } catch (err) {
      setError('Failed to fetch dashboard data')
      console.error('Dashboard fetch error:', err)
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="h-4 w-4 text-success-500" />
      case 'offline':
        return <AlertTriangle className="h-4 w-4 text-danger-500" />
      default:
        return <Clock className="h-4 w-4 text-warning-500" />
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'online':
        return <span className="badge badge-success">Online</span>
      case 'offline':
        return <span className="badge badge-danger">Offline</span>
      default:
        return <span className="badge badge-warning">Unknown</span>
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center space-x-3">
              <Shield className="h-8 w-8 text-primary-600" />
              <div>
                <h1 className="text-2xl font-bold text-gray-900">AegisFlux Console</h1>
                <p className="text-sm text-gray-500">Agent Management & Policy Builder</p>
              </div>
            </div>
            <div className="flex items-center space-x-4">
              <a href="/policy-builder" className="btn btn-primary px-4 py-2">
                Create Policy
              </a>
              <a href="/agents" className="btn btn-secondary px-4 py-2">
                <Users className="h-4 w-4 mr-2" />
                Manage Agents
              </a>
              <button className="btn btn-secondary px-4 py-2">
                <Settings className="h-4 w-4 mr-2" />
                Settings
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {error && (
          <div className="mb-6 bg-danger-50 border border-danger-200 rounded-md p-4">
            <div className="flex">
              <AlertTriangle className="h-5 w-5 text-danger-400" />
              <div className="ml-3">
                <p className="text-sm text-danger-800">{error}</p>
              </div>
            </div>
          </div>
        )}

        {/* Stats Overview */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Users className="h-8 w-8 text-primary-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Total Agents</p>
                <p className="text-2xl font-bold text-gray-900">
                  {loading ? '...' : agents.length}
                </p>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Activity className="h-8 w-8 text-success-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Online Agents</p>
                <p className="text-2xl font-bold text-gray-900">
                  {loading ? '...' : agents.filter(a => a.status === 'online').length}
                </p>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Network className="h-8 w-8 text-warning-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Active Policies</p>
                <p className="text-2xl font-bold text-gray-900">
                  {loading ? '...' : artifacts.filter(a => a.hosts && a.hosts.length > 0).length}
                </p>
              </div>
            </div>
          </div>

          <div className="card p-6">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <Server className="h-8 w-8 text-gray-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm font-medium text-gray-500">Total Artifacts</p>
                <p className="text-2xl font-bold text-gray-900">
                  {loading ? '...' : artifacts.length}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Agents Table */}
          <div className="card">
            <div className="px-6 py-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold text-gray-900">Registered Agents</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Agent
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Last Seen
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {loading ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-4 text-center text-gray-500">
                        Loading agents...
                      </td>
                    </tr>
                  ) : agents.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-4 text-center text-gray-500">
                        No agents registered
                      </td>
                    </tr>
                  ) : (
                    agents.map((agent) => (
                      <tr key={agent.agent_uid} className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center">
                            <div className="flex-shrink-0 h-10 w-10">
                              <div className="h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center">
                                <Server className="h-5 w-5 text-primary-600" />
                              </div>
                            </div>
                            <div className="ml-4">
                              <div className="text-sm font-medium text-gray-900">
                                {agent.hostname}
                              </div>
                              <div className="text-sm text-gray-500">
                                {agent.platform.primary_ip} • {agent.agent_version}
                              </div>
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center">
                            {getStatusIcon(agent.status)}
                            <span className="ml-2">{getStatusBadge(agent.status)}</span>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {formatDate(agent.last_seen)}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Policies Table */}
          <div className="card">
            <div className="px-6 py-4 border-b border-gray-200">
              <h2 className="text-lg font-semibold text-gray-900">Active Policies</h2>
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Policy
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Target
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Assigned
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {loading ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-4 text-center text-gray-500">
                        Loading policies...
                      </td>
                    </tr>
                  ) : artifacts.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-4 text-center text-gray-500">
                        No policies created
                      </td>
                    </tr>
                  ) : (
                    artifacts.map((artifact) => (
                      <tr key={artifact.id} className="hover:bg-gray-50">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div>
                            <div className="text-sm font-medium text-gray-900">
                              {artifact.name}
                            </div>
                            <div className="text-sm text-gray-500">
                              {artifact.description}
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="text-sm text-gray-900">
                            {artifact.metadata.target_ip && (
                              <span className="badge badge-info">
                                {artifact.metadata.protocol} to {artifact.metadata.target_ip}
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {artifact.hosts.length} agent{artifact.hosts.length !== 1 ? 's' : ''}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
