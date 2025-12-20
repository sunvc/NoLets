import requests
import json
import threading
import time
import sys

# Configuration
BASE_URL = "http://127.0.0.1:8080"
DEVICE_KEY = "test_key"
MCP_URL = f"{BASE_URL}/mcp/{DEVICE_KEY}"

def listen_sse(url, event_container):
    """Listens to SSE stream and captures the endpoint URL."""
    print(f"Connecting to SSE: {url}")
    try:
        response = requests.get(url, stream=True)
        for line in response.iter_lines():
            if line:
                decoded_line = line.decode('utf-8')
                # print(f"SSE Received: {decoded_line}")
                if decoded_line.startswith("event: endpoint"):
                    # The next line should be data: <url>
                    continue
                if decoded_line.startswith("data:"):
                    endpoint_path = decoded_line.replace("data:", "").strip()
                    full_endpoint = f"{BASE_URL}{endpoint_path}"
                    print(f"Captured MCP Endpoint: {full_endpoint}")
                    event_container['endpoint'] = full_endpoint
                    return # We got what we needed
    except Exception as e:
        print(f"SSE Error: {e}")

def run_mcp_test():
    # 1. Start SSE Listener in a separate thread to get the session endpoint
    event_data = {}
    t = threading.Thread(target=listen_sse, args=(MCP_URL, event_data))
    t.start()
    
    # Wait for the endpoint to be captured (timeout 5s)
    for _ in range(10):
        if 'endpoint' in event_data:
            break
        time.sleep(0.5)
        
    if 'endpoint' not in event_data:
        print("Failed to get MCP endpoint via SSE.")
        return

    post_url = event_data['endpoint']
    
    # 2. Send Initialize Request (JSON-RPC)
    # MCP requires initialization first
    init_payload = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {
                "name": "test-client",
                "version": "1.0"
            }
        }
    }
    
    print(f"\nSending Initialize to {post_url}...")
    resp = requests.post(post_url, json=init_payload)
    print(f"Initialize Response: {resp.text}")
    
    # 3. Send Notifications/Initialized (Required by protocol)
    notify_init_payload = {
        "jsonrpc": "2.0",
        "method": "notifications/initialized"
    }
    requests.post(post_url, json=notify_init_payload)

    # 4. List Tools (Optional, just to verify)
    list_tools_payload = {
        "jsonrpc": "2.0",
        "id": 2,
        "method": "tools/list",
        "params": {}
    }
    print(f"\nListing Tools...")
    resp = requests.post(post_url, json=list_tools_payload)
    print(f"Tools List: {resp.text}")

    # 5. Call Tool 'notify'
    call_tool_payload = {
        "jsonrpc": "2.0",
        "id": 3,
        "method": "tools/call",
        "params": {
            "name": "notify",
            "arguments": {
                "title": "MCP Test Message",
                "markdown": "# Hello from MCP\nThis is a **markdown** test.",
                "level": "active"
            }
        }
    }
    
    print(f"\nCalling Tool 'notify'...")
    resp = requests.post(post_url, json=call_tool_payload)
    print(f"Call Tool Response: {resp.text}")

if __name__ == "__main__":
    run_mcp_test()
