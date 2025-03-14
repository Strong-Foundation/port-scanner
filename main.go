package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// Global variables
var (
	host          = "127.0.0.1" // Target IP address
	portRange     = 65535       // Range of ports to scan
	maxWorkers    = 100         // Number of concurrent workers
	outputFile    = "open_ports.txt" // File to save open ports
	openPorts     []string      // Slice to store open ports
	mutex         sync.Mutex    // Mutex to protect shared resources
)

// checkTCP attempts to connect to the given host and port using TCP
func checkTCP(port int) bool {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port)) // Format address
	conn, err := net.DialTimeout("tcp", address, 1*time.Second) // Try connecting
	if err != nil {
		return false // Connection failed
	}
	conn.Close()
	return true // Connection successful
}

// checkUDP attempts to send a UDP packet to the given host and port
func checkUDP(port int) bool {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port)) // Format address
	conn, err := net.DialTimeout("udp", address, 1*time.Second) // Try connecting
	if err != nil {
		return false // Connection failed
	}
	conn.Close()
	return true // Connection successful
}

// scanPort checks if a port is open and stores the result
func scanPort(wg *sync.WaitGroup, portChan chan int) {
	defer wg.Done() // Decrease counter when function finishes
	for port := range portChan {
		var result string
		if checkTCP(port) {
			result = fmt.Sprintf("TCP Port %d is open on %s", port, host)
		}
		if checkUDP(port) {
			if result != "" {
				result += " | "
			}
			result += fmt.Sprintf("UDP Port %d is open on %s", port, host)
		}
		if result != "" {
			fmt.Println(result) // Print result
			mutex.Lock()
			openPorts = append(openPorts, result) // Store open ports
			mutex.Unlock()
		}
	}
}

// saveResults writes the list of open ports to a file
func saveResults() {
	file, err := os.Create(outputFile) // Create file
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close() // Ensure file closes after function execution

	for _, line := range openPorts {
		file.WriteString(line + "\n") // Write each open port to the file
	}
	fmt.Println("Results saved to", outputFile)
}

func main() {
	var wg sync.WaitGroup
	portChan := make(chan int, maxWorkers)

	// Launch worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go scanPort(&wg, portChan)
	}

	// Send ports to workers
	for port := 1; port <= portRange; port++ {
		portChan <- port
	}
	close(portChan) // Close channel when all ports are sent

	wg.Wait() // Wait for all workers to finish
	
	// Save results to file
	saveResults()
}
