# AegisFlux Console Setup Guide

This guide will help you set up and run the AegisFlux Console for agent management and policy creation.

## Prerequisites

### System Requirements
- **Node.js**: Version 18 or higher
- **npm**: Version 8 or higher (comes with Node.js)
- **Backend Services**: All AegisFlux backend services must be running

### Backend Services Required
The console requires the following backend services to be running:

| Service | Port | Purpose |
|---------|------|---------|
| Actions API | 8083 | Agent registration and management |
| BPF Registry | 8090 | Artifact storage and distribution |
| Decision Engine | 8087 | Policy decision making |
| Orchestrator | 8084 | Policy orchestration |

## Quick Start

### 1. Start Backend Services

First, ensure all backend services are running:

```bash
# From the project root directory
cd /path/to/aegisflux

# Start all backend services
docker compose -f infra/compose/docker-compose.yml up -d
```

Verify services are running:
```bash
# Check service health
curl http://localhost:8083/healthz  # Actions API
curl http://localhost:8090/healthz  # BPF Registry
curl http://localhost:8087/healthz  # Decision Engine
curl http://localhost:8084/healthz  # Orchestrator
```

### 2. Install Console Dependencies

```bash
# Navigate to console directory
cd ui/console

# Install dependencies
npm install
```

### 3. Start the Console

#### Option A: Using the startup script (recommended)
```bash
./start.sh
```

#### Option B: Manual start
```bash
npm run dev
```

### 4. Access the Console

Open your browser and navigate to: **http://127.0.0.1:3030**

## Configuration

### Environment Variables

Create a `.env.local` file in the `ui/console` directory to customize backend URLs:

```env
# Backend service URLs (optional - defaults to localhost)
NEXT_PUBLIC_ACTIONS_API_URL=http://localhost:8083
NEXT_PUBLIC_BPF_REGISTRY_URL=http://localhost:8090
NEXT_PUBLIC_ORCHESTRATOR_URL=http://localhost:8084
NEXT_PUBLIC_DECISION_API_URL=http://localhost:8087

# Debug mode (optional)
NEXT_PUBLIC_DEBUG=false
```

### Docker Environment

If running backend services in Docker, you may need to use different URLs:

```env
# For Docker Compose setup
NEXT_PUBLIC_ACTIONS_API_URL=http://host.docker.internal:8083
NEXT_PUBLIC_BPF_REGISTRY_URL=http://host.docker.internal:8090
NEXT_PUBLIC_ORCHESTRATOR_URL=http://host.docker.internal:8084
NEXT_PUBLIC_DECISION_API_URL=http://host.docker.internal:8087
```

## Features Overview

### 🖥️ Dashboard
- **System Overview**: Real-time statistics and metrics
- **Agent Status**: Quick view of registered agents
- **Policy Management**: Monitor active policies
- **Health Monitoring**: Backend service status

### 👥 Agent Management
- **Agent List**: View all registered agents
- **Agent Details**: Comprehensive system information
- **Label Management**: Organize agents with custom labels
- **Notes**: Add and manage agent notes
- **Real-time Updates**: Live status monitoring

### 🛡️ Policy Builder
- **Visual Builder**: Intuitive form-based policy creation
- **Network Rules**: Create ICMP, TCP, UDP policies
- **Target Selection**: Assign to specific agents
- **Advanced Settings**: Priority, TTL, and more
- **Instant Deployment**: Real-time policy deployment

## Usage Examples

### Viewing Agents

1. Navigate to the **Dashboard**
2. Click **Manage Agents** or go to `/agents`
3. Select any agent to view detailed information
4. Use the **Refresh** button to update status

### Creating a Network Policy

1. Click **Create Policy** from the dashboard
2. Fill in policy details:
   - **Name**: "Block ICMP to 8.8.8.8"
   - **Type**: Network Block
   - **Direction**: Egress
   - **Protocol**: ICMP
   - **Target IP**: 8.8.8.8
   - **Action**: Drop
3. Select target agents
4. Click **Create & Deploy Policy**

### Managing Agent Labels

1. Go to **Agent Management**
2. Select an agent
3. In the **Labels** section, add new labels
4. Labels help organize and filter agents

## Troubleshooting

### Common Issues

#### Backend Connection Errors
```
Error: Failed to fetch agents
```

**Solutions:**
- Verify backend services are running: `docker ps`
- Check service health: `curl http://localhost:8083/healthz`
- Verify firewall settings
- Check Docker networking if using containers

#### Agent Status Not Updating
```
Agents showing as "Unknown" status
```

**Solutions:**
- Check agent registration in Actions API
- Verify NATS connectivity
- Review agent polling configuration
- Check agent logs for errors

#### Policy Deployment Failures
```
Error: Failed to create policy
```

**Solutions:**
- Validate policy configuration
- Check agent capabilities
- Verify BPF Registry is accessible
- Review Decision Engine logs

#### Development Server Issues
```
npm run dev fails to start
```

**Solutions:**
- Check Node.js version: `node -v` (should be 18+)
- Clear node_modules: `rm -rf node_modules && npm install`
- Check port 3030 is available
- Verify package.json dependencies

### Debug Mode

Enable debug logging by setting in `.env.local`:
```env
NEXT_PUBLIC_DEBUG=true
```

This will show detailed API requests and responses in the browser console.

### Service Health Check

The startup script automatically checks backend service health. Manual check:

```bash
# Check all services
curl -s http://localhost:8083/healthz && echo " ✅ Actions API"
curl -s http://localhost:8090/healthz && echo " ✅ BPF Registry"
curl -s http://localhost:8087/healthz && echo " ✅ Decision Engine"
curl -s http://localhost:8084/healthz && echo " ✅ Orchestrator"
```

## Development

### Project Structure
```
ui/console/
├── app/                    # Next.js app directory
│   ├── page.tsx           # Dashboard
│   ├── agents/            # Agent management
│   ├── policy-builder/    # Policy creation
│   └── layout.tsx         # Root layout
├── components/            # Reusable components
├── lib/                   # API client and utilities
├── start.sh              # Startup script
└── SETUP.md              # This file
```

### Adding New Features

1. Create new pages in the `app/` directory
2. Add reusable components to `components/`
3. Update the API client in `lib/api.ts`
4. Add navigation links to existing pages
5. Update this documentation

### Building for Production

```bash
# Build the application
npm run build

# Start production server
npm start
```

## Support

For issues and questions:

1. Check the troubleshooting section above
2. Review backend service logs
3. Enable debug mode for detailed logging
4. Check the main AegisFlux documentation

## Next Steps

After setting up the console:

1. **Register Agents**: Ensure agents are registered via Actions API
2. **Create Policies**: Use the Policy Builder to create network security rules
3. **Monitor Deployment**: Track policy deployment and agent compliance
4. **Manage Agents**: Organize agents with labels and notes

The console provides a complete interface for managing your AegisFlux deployment!
