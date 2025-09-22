'use client'

import { useState, useEffect } from 'react'
import { 
  ArrowLeft, 
  Users, 
  Server, 
  Activity, 
  Settings,
  AlertTriangle,
  CheckCircle,
  Clock,
  RefreshCw,
  Tag,
  Edit,
  Trash2
} from 'lucide-react'

interface Agent {
  agent_uid: string
  org_id: string
  host_id: string
  hostname: string
  agent_version: string
  capabilities: {
    ebpf_loading: boolean
    ebpf_attach: boolean
    map_operations: boolean
    kernel_modules: string[]
    supported_hooks: string[]
    max_programs: number
    max_maps: number
  }
  platform: {
    hostname: string
    fqdn: string
    os: string
    kernel_version: string
    architecture: string
    cpu_model: string
    memory_gb: number
    disk_gb: number
    primary_ip: string
  }
  network: {
    primary_ip: string
    mac_address: string
    subnet: string
    gateway: string
    dns_servers: string[]
    ifaces: Record<string, {
      addrs: string[]
      mac: string
    }>
  }
  labels: string[]
  note: string
  created: string
  last_seen: string
  status: 'online' | 'offline' | 'unknown'
}

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)
  const [editingLabels, setEditingLabels] = useState<string>('')
  const [editingNote, setEditingNote] = useState<string>('')

  useEffect(() => {
    fetchAgents()
    const interval = setInterval(fetchAgents, 30000) // Refresh every 30 seconds
    return () => clearInterval(interval)
  }, [])

  const fetchAgents = async () => {
    try {
      setRefreshing(true)
      setError(null)

      const response = await fetch('/api/actions/agents')
      if (response.ok) {
        const data = await response.json()
        // Ensure all agents have a status property and normalize data structure
        const agentsWithStatus = (data.agents || []).map((agent: any) => ({
          ...agent,
          status: agent.status || 'unknown',
          // Normalize platform structure for different agent formats
          platform: {
            ...agent.platform,
            primary_ip: agent.platform?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            hostname: agent.platform?.hostname || agent.hostname,
            os: agent.platform?.os || 'linux',
            kernel_version: agent.platform?.kernel_version || 'unknown',
            architecture: agent.platform?.architecture || agent.platform?.arch || 'unknown',
            fqdn: agent.platform?.fqdn || agent.hostname,
            cpu_model: agent.platform?.cpu_model || 'Unknown',
            memory_gb: agent.platform?.memory_gb || 0,
            disk_gb: agent.platform?.disk_gb || 0
          },
          // Normalize network structure
          network: {
            ...agent.network,
            primary_ip: agent.network?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            mac_address: agent.network?.mac_address || agent.network?.ifaces?.ens160?.mac || 'unknown',
            subnet: agent.network?.subnet || agent.network?.addrs?.[0] || 'unknown',
            gateway: agent.network?.gateway || 'unknown',
            dns_servers: agent.network?.dns_servers || []
          },
          // Add default capabilities if missing
          capabilities: agent.capabilities || {
            ebpf_loading: true,
            ebpf_attach: true,
            map_operations: true,
            kernel_modules: ['bpf'],
            supported_hooks: ['tc', 'xdp'],
            max_programs: 10,
            max_maps: 50
          }
        }))
        // Sort agents by last_seen (most recent first) and group by host_id
        const sortedAgents = agentsWithStatus.sort((a: any, b: any) => 
          new Date(b.last_seen).getTime() - new Date(a.last_seen).getTime()
        )
        setAgents(sortedAgents)
      } else {
        setError('Failed to fetch agents')
      }
    } catch (err) {
      setError('Failed to fetch agents')
      console.error('Agent fetch error:', err)
    } finally {
      setLoading(false)
      setRefreshing(false)
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

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const updateAgentLabels = async (agentUid: string, labels: string[]) => {
    try {
      const response = await fetch(`/api/actions/agents/${agentUid}/labels`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          add: labels.filter(label => !agents.find(a => a.agent_uid === agentUid)?.labels.includes(label)),
          remove: agents.find(a => a.agent_uid === agentUid)?.labels.filter(label => !labels.includes(label)) || []
        })
      })

      if (response.ok) {
        await fetchAgents()
        setEditingLabels('')
      }
    } catch (err) {
      console.error('Failed to update labels:', err)
    }
  }

  const updateAgentNote = async (agentUid: string, note: string) => {
    try {
      const response = await fetch(`/api/actions/agents/${agentUid}/note`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ note })
      })

      if (response.ok) {
        await fetchAgents()
        setEditingNote('')
      }
    } catch (err) {
      console.error('Failed to update note:', err)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between py-6">
            <div className="flex items-center space-x-4">
              <a href="/" className="flex items-center text-gray-600 hover:text-gray-900">
                <ArrowLeft className="h-5 w-5 mr-2" />
                Back to Dashboard
              </a>
              <div className="h-6 w-px bg-gray-300" />
              <div className="flex items-center space-x-3">
                <Users className="h-8 w-8 text-primary-600" />
                <div>
                  <h1 className="text-2xl font-bold text-gray-900">Agent Management</h1>
                  <p className="text-sm text-gray-500">Monitor and manage registered agents</p>
                </div>
              </div>
            </div>
            <button
              onClick={fetchAgents}
              disabled={refreshing}
              className="btn btn-secondary px-4 py-2"
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
              Refresh
            </button>
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

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Agents List */}
          <div className="lg:col-span-2">
            <div className="card">
              <div className="px-6 py-4 border-b border-gray-200">
                <h2 className="text-lg font-semibold text-gray-900">
                  Registered Agents ({agents.length})
                </h2>
              </div>
              <div className="divide-y divide-gray-200">
                {loading ? (
                  <div className="p-6 text-center text-gray-500">
                    Loading agents...
                  </div>
                ) : agents.length === 0 ? (
                  <div className="p-6 text-center text-gray-500">
                    No agents registered
                  </div>
                ) : (
                  agents.map((agent) => (
                    <div
                      key={agent.agent_uid}
                      className={`p-6 hover:bg-gray-50 cursor-pointer transition-colors ${
                        selectedAgent?.agent_uid === agent.agent_uid ? 'bg-primary-50 border-r-4 border-primary-500' : ''
                      }`}
                      onClick={() => setSelectedAgent(agent)}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex items-start space-x-4">
                          <div className="flex-shrink-0">
                            <div className="h-12 w-12 rounded-full bg-primary-100 flex items-center justify-center">
                              <Server className="h-6 w-6 text-primary-600" />
                            </div>
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center space-x-2">
                              <h3 className="text-lg font-medium text-gray-900 truncate">
                                {agent.hostname}
                              </h3>
                              {getStatusIcon(agent.status)}
                              {getStatusBadge(agent.status)}
                            </div>
                            <p className="text-sm text-gray-500">
                              {agent.platform.primary_ip} • {agent.agent_version}
                            </p>
                            <p className="text-sm text-gray-500">
                              {agent.platform.os} • {agent.platform.architecture}
                            </p>
                            <div className="mt-2 flex flex-wrap gap-1">
                              {agent.labels.map((label) => (
                                <span key={label} className="badge badge-info">
                                  {label}
                                </span>
                              ))}
                            </div>
                          </div>
                        </div>
                        <div className="text-right text-sm text-gray-500">
                          <p>Last seen: {formatDate(agent.last_seen)}</p>
                          <p>Created: {formatDate(agent.created)}</p>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          </div>

          {/* Agent Details */}
          <div className="lg:col-span-1">
            {selectedAgent ? (
              <div className="space-y-6">
                {/* Basic Info */}
                <div className="card p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Agent Details</h3>
                  <dl className="space-y-3">
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Host ID</dt>
                      <dd className="text-sm text-gray-900 font-mono">{selectedAgent.host_id}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Agent UID</dt>
                      <dd className="text-sm text-gray-900 font-mono break-all">{selectedAgent.agent_uid}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Organization</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.org_id}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">FQDN</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.platform.fqdn}</dd>
                    </div>
                  </dl>
                </div>

                {/* System Info */}
                <div className="card p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">System Information</h3>
                  <dl className="space-y-3">
                    <div>
                      <dt className="text-sm font-medium text-gray-500">CPU</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.platform.cpu_model}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Memory</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.platform.memory_gb} GB</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Disk</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.platform.disk_gb} GB</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Kernel</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.platform.kernel_version}</dd>
                    </div>
                  </dl>
                </div>

                {/* Network Info */}
                <div className="card p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Network Configuration</h3>
                  <dl className="space-y-3">
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Primary IP</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.network.primary_ip}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">MAC Address</dt>
                      <dd className="text-sm text-gray-900 font-mono">{selectedAgent.network.mac_address}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Subnet</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.network.subnet}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Gateway</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.network.gateway}</dd>
                    </div>
                  </dl>
                </div>

                {/* Capabilities */}
                <div className="card p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">eBPF Capabilities</h3>
                  <dl className="space-y-3">
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Max Programs</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.capabilities.max_programs}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Max Maps</dt>
                      <dd className="text-sm text-gray-900">{selectedAgent.capabilities.max_maps}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Supported Hooks</dt>
                      <dd className="text-sm text-gray-900">
                        <div className="flex flex-wrap gap-1 mt-1">
                          {selectedAgent.capabilities.supported_hooks.map((hook) => (
                            <span key={hook} className="badge badge-info">
                              {hook}
                            </span>
                          ))}
                        </div>
                      </dd>
                    </div>
                  </dl>
                </div>

                {/* Labels Management */}
                <div className="card p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Labels</h3>
                  <div className="space-y-3">
                    <div className="flex flex-wrap gap-1">
                      {selectedAgent.labels.map((label) => (
                        <span key={label} className="badge badge-info">
                          {label}
                        </span>
                      ))}
                    </div>
                    <div className="flex space-x-2">
                      <input
                        type="text"
                        value={editingLabels}
                        onChange={(e) => setEditingLabels(e.target.value)}
                        placeholder="Add label (comma-separated)"
                        className="input flex-1"
                      />
                      <button
                        onClick={() => {
                          const newLabels = editingLabels.split(',').map(l => l.trim()).filter(l => l)
                          updateAgentLabels(selectedAgent.agent_uid, [...selectedAgent.labels, ...newLabels])
                        }}
                        className="btn btn-primary px-3 py-2"
                      >
                        <Tag className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>

                {/* Notes */}
                <div className="card p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">Notes</h3>
                  <div className="space-y-3">
                    <p className="text-sm text-gray-900">{selectedAgent.note || 'No notes'}</p>
                    <div className="flex space-x-2">
                      <input
                        type="text"
                        value={editingNote}
                        onChange={(e) => setEditingNote(e.target.value)}
                        placeholder="Add or update note"
                        className="input flex-1"
                      />
                      <button
                        onClick={() => updateAgentNote(selectedAgent.agent_uid, editingNote)}
                        className="btn btn-primary px-3 py-2"
                      >
                        <Edit className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <div className="card p-6">
                <div className="text-center text-gray-500">
                  <Users className="h-12 w-12 mx-auto mb-4 text-gray-400" />
                  <p>Select an agent to view details</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  )
}
