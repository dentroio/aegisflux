import { CheckCircle, AlertTriangle, Clock, Server } from 'lucide-react'

interface AgentStatusProps {
  status: 'online' | 'offline' | 'unknown'
  hostname: string
  ip: string
  lastSeen: string
  className?: string
}

export function AgentStatus({ status, hostname, ip, lastSeen, className = '' }: AgentStatusProps) {
  const getStatusIcon = () => {
    switch (status) {
      case 'online':
        return <CheckCircle className="h-4 w-4 text-success-500" />
      case 'offline':
        return <AlertTriangle className="h-4 w-4 text-danger-500" />
      default:
        return <Clock className="h-4 w-4 text-warning-500" />
    }
  }

  const getStatusBadge = () => {
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

  return (
    <div className={`flex items-center space-x-3 ${className}`}>
      <div className="flex-shrink-0">
        <div className="h-10 w-10 rounded-full bg-primary-100 flex items-center justify-center">
          <Server className="h-5 w-5 text-primary-600" />
        </div>
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center space-x-2">
          <p className="text-sm font-medium text-gray-900 truncate">{hostname}</p>
          {getStatusIcon()}
          {getStatusBadge()}
        </div>
        <p className="text-sm text-gray-500 truncate">{ip}</p>
        <p className="text-xs text-gray-400">Last seen: {formatDate(lastSeen)}</p>
      </div>
    </div>
  )
}
