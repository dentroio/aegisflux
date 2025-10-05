# Backend Documentation

This directory contains documentation for backend services and implementation.

## 📁 Files

### **Backend Architecture**
- `BACKEND_SPECIFICATION_FOR_AGENT.md` - Complete backend API specification for agent integration
- `BACKEND_STORAGE_ISSUES_ANALYSIS.md` - Analysis of backend storage capabilities
- `BACKEND_WEBSOCKET_IMPLEMENTATION_PLAN.md` - WebSocket gateway implementation plan

### **Implementation**
- `BACKEND_IMPLEMENTATION_CHECKLIST.md` - Checklist for backend implementation tasks

## 🏗️ Backend Services

### **Core Services**
- **Actions API** (`backend/actions-api/`) - Agent registration and management
- **BPF Registry** (`backend/bpf-registry/`) - eBPF artifact storage and distribution
- **Decision Service** (`backend/decision/`) - Policy decision engine
- **Orchestrator** (`backend/orchestrator/`) - Policy orchestration and deployment

### **New Services**
- **WebSocket Gateway** (`backend/websocket-gateway/`) - Real-time agent communication
- **Config API** (`backend/config-api/`) - Configuration management
- **Correlator** (`backend/correlator/`) - Event correlation and analysis

## 🔗 Integration

- **Agent Integration**: See `docs/agents/` for agent-side integration guides
- **API Documentation**: See individual service README files in `backend/*/README.md`
- **Testing**: See `tests/` directory for integration and end-to-end tests