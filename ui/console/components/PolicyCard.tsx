import { Shield, Network, Target, Clock, Users } from 'lucide-react'

interface PolicyCardProps {
  id: string
  name: string
  description: string
  policyType: string
  targetIp?: string
  protocol?: string
  direction?: string
  action: string
  assignedHosts: number
  createdAt: string
  size: number
  className?: string
  onClick?: () => void
}

export function PolicyCard({
  id,
  name,
  description,
  policyType,
  targetIp,
  protocol,
  direction,
  action,
  assignedHosts,
  createdAt,
  size,
  className = '',
  onClick
}: PolicyCardProps) {
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes'
    const k = 1024
    const sizes = ['Bytes', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString()
  }

  const getActionBadge = (action: string) => {
    switch (action.toLowerCase()) {
      case 'drop':
        return <span className="badge badge-danger">Drop</span>
      case 'allow':
        return <span className="badge badge-success">Allow</span>
      case 'log':
        return <span className="badge badge-info">Log</span>
      default:
        return <span className="badge badge-warning">{action}</span>
    }
  }

  const getDirectionBadge = (direction?: string) => {
    if (!direction) return null
    
    switch (direction.toLowerCase()) {
      case 'egress':
        return <span className="badge badge-info">Egress</span>
      case 'ingress':
        return <span className="badge badge-warning">Ingress</span>
      case 'both':
        return <span className="badge badge-info">Both</span>
      default:
        return <span className="badge badge-warning">{direction}</span>
    }
  }

  return (
    <div
      className={`card p-6 hover:shadow-md transition-shadow cursor-pointer ${className}`}
      onClick={onClick}
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center space-x-3">
          <div className="flex-shrink-0">
            <div className="h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center">
              <Shield className="h-5 w-5 text-primary-600" />
            </div>
          </div>
          <div>
            <h3 className="text-lg font-semibold text-gray-900">{name}</h3>
            <p className="text-sm text-gray-500">{description}</p>
          </div>
        </div>
        {getActionBadge(action)}
      </div>

      <div className="space-y-3">
        {/* Policy Type */}
        <div className="flex items-center space-x-2">
          <Network className="h-4 w-4 text-gray-400" />
          <span className="text-sm text-gray-600">Type:</span>
          <span className="text-sm font-medium text-gray-900 capitalize">
            {policyType.replace('_', ' ')}
          </span>
        </div>

        {/* Target Configuration */}
        {targetIp && (
          <div className="flex items-center space-x-2">
            <Target className="h-4 w-4 text-gray-400" />
            <span className="text-sm text-gray-600">Target:</span>
            <span className="text-sm font-medium text-gray-900">{targetIp}</span>
            {protocol && (
              <span className="badge badge-info">{protocol.toUpperCase()}</span>
            )}
            {getDirectionBadge(direction)}
          </div>
        )}

        {/* Assignment Info */}
        <div className="flex items-center justify-between pt-3 border-t border-gray-100">
          <div className="flex items-center space-x-2">
            <Users className="h-4 w-4 text-gray-400" />
            <span className="text-sm text-gray-600">
              {assignedHosts} agent{assignedHosts !== 1 ? 's' : ''}
            </span>
          </div>
          <div className="flex items-center space-x-4 text-xs text-gray-500">
            <div className="flex items-center space-x-1">
              <Clock className="h-3 w-3" />
              <span>{formatDate(createdAt)}</span>
            </div>
            <span>{formatBytes(size)}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
