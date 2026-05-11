'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { useForm } from 'react-hook-form'
import { ConsoleShell } from '@/components/shell/ConsoleShell'
import { readLabAuthenticated } from '@/shared/labAuth'
import {
  Shield,
  Play,
  Save,
  AlertCircle,
  CheckCircle,
} from 'lucide-react'

interface Agent {
  agent_uid: string
  host_id: string
  hostname: string
  platform: {
    primary_ip: string
  }
  network: {
    primary_ip: string
  }
  status: string
}

interface PolicyForm {
  name: string
  description: string
  policy_type: 'network_block' | 'network_allow' | 'system_call_block' | 'file_access_block'
  direction: 'ingress' | 'egress' | 'both'
  protocol: 'icmp' | 'tcp' | 'udp' | 'all'
  target_ip: string
  target_port?: string
  target_hosts: string[]
  action: 'drop' | 'allow' | 'log'
  priority: number
  ttl: number
}

export default function PolicyBuilder() {
  const router = useRouter()
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [success, setSuccess] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [gate, setGate] = useState(false)

  useEffect(() => {
    if (!readLabAuthenticated()) { router.replace('/'); return }
    setGate(true)
  }, [router])

  const { register, handleSubmit, watch, setValue, formState: { errors } } = useForm<PolicyForm>({
    defaultValues: {
      name: '',
      description: '',
      policy_type: 'network_block',
      direction: 'egress',
      protocol: 'icmp',
      target_ip: '',
      target_port: '',
      target_hosts: [],
      action: 'drop',
      priority: 100,
      ttl: 3600
    }
  })


  useEffect(() => {
    fetchAgents()
  }, [])

  const fetchAgents = async () => {
    try {
      setLoading(true)
      const response = await fetch('/api/actions/agents')
      if (response.ok) {
        const data = await response.json()
        // Ensure all agents have required properties and normalize data structure
        const agentsWithDefaults = (data.agents || []).map((agent: any) => ({
          ...agent,
          status: agent.status || 'unknown',
          platform: {
            ...agent.platform,
            primary_ip: agent.platform?.primary_ip || agent.network?.addrs?.[0]?.split('/')[0] || 'unknown',
            hostname: agent.platform?.hostname || agent.hostname,
            os: agent.platform?.os || 'linux',
            kernel_version: agent.platform?.kernel_version || 'unknown',
            architecture: agent.platform?.architecture || agent.platform?.arch || 'unknown'
          }
        }))
        setAgents(agentsWithDefaults)
      }
    } catch (err) {
      setError('Failed to fetch agents')
      console.error('Agent fetch error:', err)
    } finally {
      setLoading(false)
    }
  }

  const onSubmit = async (data: PolicyForm) => {
    try {
      setCreating(true)
      setError(null)
      setSuccess(null)

      // Create the policy artifact
      const policyResponse = await fetch('/api/decision/plans/policy', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          control_intents: [{
            name: data.name,
            description: data.description,
            type: data.policy_type,
            direction: data.direction,
            protocol: data.protocol,
            target_ip: data.target_ip,
            target_port: data.target_port,
            action: data.action,
            priority: data.priority,
            ttl: data.ttl
          }]
        })
      })

      if (!policyResponse.ok) {
        throw new Error('Failed to create policy')
      }

      const policyData = await policyResponse.json()
      
      // Create artifact in BPF Registry
      const artifactResponse = await fetch('/api/registry/artifacts', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: data.name,
          version: '1.0.0',
          description: data.description,
          type: 'program',
          architecture: 'x86_64',
          kernel_version: '5.4.0',
          metadata: {
            policy_type: data.policy_type,
            direction: data.direction,
            protocol: data.protocol,
            target_ip: data.target_ip,
            target_port: data.target_port,
            action: data.action,
            priority: data.priority,
            ttl: data.ttl
          },
          tags: ['network', 'security'],
          data: btoa('placeholder_artifact_data') // This would be the actual tar.zst data
        })
      })

      if (!artifactResponse.ok) {
        throw new Error('Failed to create artifact')
      }

      const artifactData = await artifactResponse.json()

      // Assign to selected hosts
      for (const hostId of data.target_hosts) {
        const assignResponse = await fetch(`/api/registry/assign/${artifactData.id}/${hostId}`, {
          method: 'POST'
        })

        if (!assignResponse.ok) {
          console.warn(`Failed to assign policy to host ${hostId}`)
        }
      }

      setSuccess(`Policy "${data.name}" created and deployed successfully!`)
      // Reset form using the form methods from useForm
      // Note: form.reset() would be available if we destructured it from useForm
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create policy')
      console.error('Policy creation error:', err)
    } finally {
      setCreating(false)
    }
  }

  const handleHostToggle = (hostId: string) => {
    const currentHosts = watch('target_hosts')
    const newHosts = currentHosts.includes(hostId)
      ? currentHosts.filter(id => id !== hostId)
      : [...currentHosts, hostId]
    setValue('target_hosts', newHosts)
  }

  const policyType = watch('policy_type')
  const protocol = watch('protocol')

  if (!gate) return <div className="flex min-h-screen items-center justify-center text-sm text-gray-500">Loading…</div>

  return (
    <ConsoleShell
      activeNavId="policy-builder"
      breadcrumbs={[{ label: 'Policy Builder' }]}
      health={{ label: 'Enforcement', tone: 'amber', text: 'Policies can block traffic' }}
      onLogout={() => { window.localStorage.removeItem('aegisflux.labAuth'); router.replace('/') }}
    >
      <main className="mx-auto max-w-4xl px-5 py-6">
        {/* Success/Error Messages */}
        {success && (
          <div className="mb-6 bg-success-50 border border-success-200 rounded-md p-4">
            <div className="flex">
              <CheckCircle className="h-5 w-5 text-success-400" />
              <div className="ml-3">
                <p className="text-sm text-success-800">{success}</p>
              </div>
            </div>
          </div>
        )}

        {error && (
          <div className="mb-6 bg-danger-50 border border-danger-200 rounded-md p-4">
            <div className="flex">
              <AlertCircle className="h-5 w-5 text-danger-400" />
              <div className="ml-3">
                <p className="text-sm text-danger-800">{error}</p>
              </div>
            </div>
          </div>
        )}

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-8">
          {/* Policy Information */}
          <div className="card p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Policy Information</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Policy Name *
                </label>
                <input
                  {...register('name', { required: 'Policy name is required' })}
                  className="input w-full"
                  placeholder="e.g., Block ICMP to 8.8.8.8"
                />
                {errors.name && (
                  <p className="mt-1 text-sm text-danger-600">{errors.name.message}</p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Policy Type *
                </label>
                <select
                  {...register('policy_type')}
                  className="input w-full"
                >
                  <option value="network_block">Network Block</option>
                  <option value="network_allow">Network Allow</option>
                  <option value="system_call_block">System Call Block</option>
                  <option value="file_access_block">File Access Block</option>
                </select>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Description
                </label>
                <textarea
                  {...register('description')}
                  rows={3}
                  className="input w-full"
                  placeholder="Describe what this policy does..."
                />
              </div>
            </div>
          </div>

          {/* Network Configuration */}
          <div className="card p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Network Configuration</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Direction *
                </label>
                <select
                  {...register('direction')}
                  className="input w-full"
                >
                  <option value="egress">Egress (Outbound)</option>
                  <option value="ingress">Ingress (Inbound)</option>
                  <option value="both">Both Directions</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Protocol *
                </label>
                <select
                  {...register('protocol')}
                  className="input w-full"
                >
                  <option value="icmp">ICMP</option>
                  <option value="tcp">TCP</option>
                  <option value="udp">UDP</option>
                  <option value="all">All Protocols</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Target IP Address *
                </label>
                <input
                  {...register('target_ip', { 
                    required: 'Target IP is required',
                    pattern: {
                      value: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
                      message: 'Invalid IP address'
                    }
                  })}
                  className="input w-full"
                  placeholder="e.g., 8.8.8.8"
                />
                {errors.target_ip && (
                  <p className="mt-1 text-sm text-danger-600">{errors.target_ip.message}</p>
                )}
              </div>

              {protocol === 'tcp' || protocol === 'udp' ? (
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Target Port
                  </label>
                  <input
                    {...register('target_port')}
                    type="number"
                    min="1"
                    max="65535"
                    className="input w-full"
                    placeholder="e.g., 80, 443"
                  />
                </div>
              ) : null}

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Action *
                </label>
                <select
                  {...register('action')}
                  className="input w-full"
                >
                  <option value="drop">Drop</option>
                  <option value="allow">Allow</option>
                  <option value="log">Log Only</option>
                </select>
              </div>
            </div>
          </div>

          {/* Target Agents */}
          <div className="card p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Target Agents</h2>
            {loading ? (
              <p className="text-gray-500">Loading agents...</p>
            ) : agents.length === 0 ? (
              <p className="text-gray-500">No agents available</p>
            ) : (
              <div className="space-y-3">
                {agents.map((agent) => (
                  <label key={agent.agent_uid} className="flex items-center space-x-3">
                    <input
                      type="checkbox"
                      checked={watch('target_hosts').includes(agent.host_id)}
                      onChange={() => handleHostToggle(agent.host_id)}
                      className="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
                    />
                    <div className="flex-1">
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-gray-900">
                          {agent.hostname}
                        </span>
                        <span className="text-sm text-gray-500">
                          {agent.platform.primary_ip}
                        </span>
                      </div>
                      <div className="text-sm text-gray-500">
                        {agent.host_id}
                      </div>
                    </div>
                  </label>
                ))}
              </div>
            )}
          </div>

          {/* Advanced Settings */}
          <div className="card p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Advanced Settings</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Priority
                </label>
                <input
                  {...register('priority', { 
                    valueAsNumber: true,
                    min: { value: 1, message: 'Priority must be at least 1' },
                    max: { value: 1000, message: 'Priority must be at most 1000' }
                  })}
                  type="number"
                  min="1"
                  max="1000"
                  className="input w-full"
                />
                {errors.priority && (
                  <p className="mt-1 text-sm text-danger-600">{errors.priority.message}</p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  TTL (seconds)
                </label>
                <input
                  {...register('ttl', { 
                    valueAsNumber: true,
                    min: { value: 60, message: 'TTL must be at least 60 seconds' }
                  })}
                  type="number"
                  min="60"
                  className="input w-full"
                />
                {errors.ttl && (
                  <p className="mt-1 text-sm text-danger-600">{errors.ttl.message}</p>
                )}
              </div>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end space-x-4">
            <button
              type="button"
              className="btn btn-secondary px-6 py-2"
              onClick={() => window.location.reload()}
            >
              <Save className="h-4 w-4 mr-2" />
              Reset
            </button>
            <button
              type="submit"
              disabled={creating}
              className="btn btn-primary px-6 py-2"
            >
              {creating ? (
                <>
                  <div className="animate-spin h-4 w-4 mr-2 border-2 border-white border-t-transparent rounded-full" />
                  Creating...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  Create & Deploy Policy
                </>
              )}
            </button>
          </div>
        </form>
      </main>
    </ConsoleShell>
  )
}
