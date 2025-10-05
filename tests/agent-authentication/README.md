# Agent Authentication Tests

This directory contains test scripts for agent authentication and WebSocket communication.

## 🧪 Test Scripts

### **Working Examples**
- `agent_authentication_test.py` - Complete working authentication test
  - Demonstrates proper authentication flow
  - Shows correct SecureMessage format
  - Includes heartbeat and status messages
  - **Status**: ✅ Working and tested

### **Diagnostic Tools**
- `diagnostic_agent_test.py` - Diagnostic test showing correct vs incorrect approaches
  - Test 1: Wrong approach (sending messages without auth) - Should fail
  - Test 2: Correct approach (proper auth flow) - Should succeed  
  - Test 3: Wrong message format - Should fail
  - **Purpose**: Helps identify authentication issues

## 🚀 Usage

### **Run Working Test**
```bash
cd tests/agent-authentication
python3 agent_authentication_test.py
```

### **Run Diagnostic Test**
```bash
cd tests/agent-authentication
python3 diagnostic_agent_test.py
```

## 📋 Prerequisites

- Backend WebSocket Gateway running on `ws://localhost:8080/ws/agent`
- Python 3.7+ with required packages:
  - `websockets`
  - `cryptography`

## 🔍 Expected Results

### **Working Test**
- ✅ WebSocket connection established
- ✅ Authentication successful
- ✅ Heartbeat messages sent and received
- ✅ Agent status messages sent

### **Diagnostic Test**
- ❌ Test 1: "agent not authenticated" error
- ✅ Test 2: "Authentication SUCCESSFUL!"
- ❌ Test 3: "agent not authenticated" error

## 📚 Related Documentation

- **Authentication Guide**: `docs/agents/AGENT_AUTHENTICATION_EXAMPLE.md`
- **Troubleshooting**: `docs/agents/AGENT_AUTHENTICATION_DIAGNOSTIC.md`
- **Quick Reference**: `docs/agents/AGENT_QUICK_REFERENCE.md`