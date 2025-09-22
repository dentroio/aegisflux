#!/usr/bin/env python3
"""
Simple script to assign artifacts to hosts in the BPF registry.
This is a temporary solution until proper host assignment APIs are implemented.
"""

import requests
import json
import sys

REGISTRY_URL = "http://localhost:8090"

def get_artifacts():
    """Get all available artifacts from the registry."""
    try:
        response = requests.get(f"{REGISTRY_URL}/artifacts")
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"Error fetching artifacts: {e}")
        return None

def assign_artifact_to_host(artifact_id, host_id):
    """Assign an artifact to a host by updating the artifact's hosts field.
    Note: This is a temporary solution - ideally there should be a proper API endpoint.
    """
    print(f"Note: This script demonstrates the concept. The BPF registry would need")
    print(f"an API endpoint to assign artifacts to hosts (e.g., POST /artifacts/{id}/hosts)")
    print(f"")
    print(f"To assign artifact {artifact_id} to host {host_id}, the registry would need:")
    print(f"1. An API endpoint like: POST /artifacts/{artifact_id}/hosts")
    print(f"2. Or a bulk assignment endpoint: POST /artifacts/assign")
    print(f"3. The request body would be: {{'host_ids': ['{host_id}']}}")
    print(f"")
    print(f"For now, we can verify the current state:")
    
    # Check current artifacts for the host
    response = requests.get(f"{REGISTRY_URL}/artifacts/for-host/{host_id}")
    print(f"Current artifacts for {host_id}: {response.json()}")

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 assign_artifacts_to_host.py <artifact_id> <host_id>")
        print("Example: python3 assign_artifacts_to_host.py artifact_123 linux-host")
        sys.exit(1)
    
    artifact_id = sys.argv[1]
    host_id = sys.argv[2]
    
    print(f"=== Assigning Artifact to Host ===")
    print(f"Artifact ID: {artifact_id}")
    print(f"Host ID: {host_id}")
    print()
    
    # Get available artifacts
    artifacts_data = get_artifacts()
    if not artifacts_data:
        sys.exit(1)
    
    print("Available artifacts:")
    for artifact in artifacts_data.get('artifacts', []):
        print(f"  - {artifact['id']}: {artifact['name']} v{artifact['version']}")
    
    print()
    
    # Check if the artifact exists
    artifact_exists = any(art['id'] == artifact_id for art in artifacts_data.get('artifacts', []))
    if not artifact_exists:
        print(f"Error: Artifact {artifact_id} not found!")
        print("Available artifact IDs are listed above.")
        sys.exit(1)
    
    # Demonstrate the assignment concept
    assign_artifact_to_host(artifact_id, host_id)

if __name__ == "__main__":
    main()