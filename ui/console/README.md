# AegisFlux Console

A modern web-based management console for AegisFlux agent management and network security policy creation.

## Features

### 🖥️ Agent Management
- **Real-time Agent Monitoring**: View all registered agents with live status updates
- **Agent Details**: Comprehensive system information, capabilities, and network configuration
- **Label Management**: Organize agents with custom labels and notes
- **Health Monitoring**: Track agent connectivity and last seen timestamps

### 🛡️ Policy Builder
- **Visual Policy Creation**: Intuitive form-based policy builder
- **Network Security Rules**: Create ICMP, TCP, UDP blocking/allow rules
- **Target Selection**: Assign policies to specific agents or groups
- **Real-time Deployment**: Instant policy deployment to selected agents

### 📊 Dashboard
- **System Overview**: Key metrics and statistics
- **Agent Status**: Quick view of agent health and policy assignments
- **Policy Management**: Monitor active policies and their enforcement

## Technology Stack

- **Frontend**: Next.js 14 with React 18
- **Styling**: Tailwind CSS with custom design system
- **Icons**: Lucide React
- **Forms**: React Hook Form
- **Data Fetching**: Native fetch with React Query integration
- **TypeScript**: Full type safety throughout

## Getting Started

### Prerequisites

- Node.js 18+ 
- npm or yarn
- Running AegisFlux backend services

### Installation

```bash
# Navigate to the console directory
cd ui/console

# Install dependencies
npm install

# Start development server
npm run dev
```

The console will be available at `http://localhost:3030`

### Environment Configuration

Create a `.env.local` file in the console directory:

```env
# Backend service URLs (optional - defaults to localhost)
NEXT_PUBLIC_ACTIONS_API_URL=http://localhost:8083
NEXT_PUBLIC_BPF_REGISTRY_URL=http://localhost:8090
NEXT_PUBLIC_ORCHESTRATOR_URL=http://localhost:8084
NEXT_PUBLIC_DECISION_API_URL=http://localhost:8087
```

## API Integration

The console integrates with the following AegisFlux backend services:

### Actions API (Port 8083)
- Agent registration and management
- Agent metadata updates
- Health monitoring

### BPF Registry (Port 8090)
- Artifact storage and retrieval
- Host assignment management
- Policy distribution

### Decision Engine (Port 8087)
- Policy intent processing
- Control generation
- Plan management

### Orchestrator (Port 8084)
- Policy orchestration
- Artifact compilation
- Deployment coordination

## Usage

### Viewing Agents

1. Navigate to the **Dashboard** to see an overview of all registered agents
2. Click on **Agent Management** for detailed agent information
3. Select any agent to view comprehensive system details
4. Use the **Refresh** button to update agent status in real-time

### Creating Policies

1. Click **Create Policy** from the dashboard or navigate to **Policy Builder**
2. Fill in the policy information:
   - **Policy Name**: Descriptive name for the policy
   - **Policy Type**: Network block/allow, system call block, etc.
   - **Description**: Optional detailed description
3. Configure network settings:
   - **Direction**: Ingress, egress, or both
   - **Protocol**: ICMP, TCP, UDP, or all
   - **Target IP**: IP address to apply the policy to
   - **Target Port**: Port number (for TCP/UDP)
   - **Action**: Drop, allow, or log
4. Select target agents from the registered agents list
5. Configure advanced settings (priority, TTL)
6. Click **Create & Deploy Policy** to deploy

### Managing Policies

- View active policies in the dashboard
- Monitor policy assignments and enforcement
- Track policy effectiveness and agent compliance

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Next.js UI    │────│   API Proxy     │────│  Backend APIs   │
│                 │    │   (Next.js)     │    │                 │
│ • Dashboard     │    │ • /api/actions  │    │ • Actions API   │
│ • Agent Mgmt    │    │ • /api/registry │    │ • BPF Registry  │
│ • Policy Builder│    │ • /api/decision │    │ • Decision API  │
│ • Real-time     │    │ • /api/orchestr │    │ • Orchestrator  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Development

### Project Structure

```
ui/console/
├── app/                    # Next.js app directory
│   ├── page.tsx           # Dashboard homepage
│   ├── agents/            # Agent management page
│   ├── policy-builder/    # Policy creation page
│   ├── layout.tsx         # Root layout
│   └── globals.css        # Global styles
├── components/            # Reusable UI components
│   ├── AgentStatus.tsx    # Agent status display
│   └── PolicyCard.tsx     # Policy information card
├── lib/                   # Utilities and API client
│   └── api.ts            # Backend API integration
├── package.json          # Dependencies and scripts
├── tailwind.config.js    # Tailwind configuration
├── next.config.js        # Next.js configuration
└── tsconfig.json         # TypeScript configuration
```

### Key Components

- **Dashboard**: Main overview with statistics and recent activity
- **AgentStatus**: Displays agent information with status indicators
- **PolicyCard**: Shows policy details with assignment information
- **API Client**: Centralized backend service integration

### Styling

The console uses a custom Tailwind CSS design system with:
- Consistent color palette (primary, success, warning, danger)
- Reusable component classes (btn, card, input, badge)
- Responsive design patterns
- Dark/light theme support ready

## Deployment

### Production Build

```bash
npm run build
npm start
```

### Docker Deployment

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build
EXPOSE 3030
CMD ["npm", "start"]
```

## Troubleshooting

### Common Issues

1. **Backend Connection Errors**
   - Verify backend services are running
   - Check environment variables
   - Confirm API endpoints are accessible

2. **Agent Status Not Updating**
   - Check agent registration status
   - Verify NATS connectivity
   - Review agent polling configuration

3. **Policy Deployment Failures**
   - Validate policy configuration
   - Check agent capabilities
   - Review backend service logs

### Debug Mode

Enable debug logging by setting:
```env
NEXT_PUBLIC_DEBUG=true
```

## Contributing

1. Follow the existing code style and patterns
2. Add TypeScript types for all new features
3. Include responsive design considerations
4. Test with real backend services
5. Update documentation for new features

## License

This project is part of the AegisFlux system and follows the same licensing terms.
