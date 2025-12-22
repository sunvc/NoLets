package push

import (
	"log"

	"golang.org/x/net/http2"
)

// CloseAPNSClients closes all APNS client resources
func CloseAPNSClients() {
	// Close channel and clean up resources
	if CLIENTS != nil {
		// Try to close all client connections
		clientCount := len(CLIENTS)
		for i := 0; i < clientCount; i++ {
			select {
			case client := <-CLIENTS:
				// If the client has resources that need to be closed specially, handle them here
				// For example, close the HTTP client connection pool
				if client != nil && client.HTTPClient != nil && client.HTTPClient.Transport != nil {
					// Try to close transport
					if transport, ok := client.HTTPClient.Transport.(*http2.Transport); ok && transport != nil {
						transport.CloseIdleConnections()
					}
				}
			default:
				// Channel is empty
				break
			}
		}

		// Log close information
		log.Println("All APNS clients have been closed")
	}
}
