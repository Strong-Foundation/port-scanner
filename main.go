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
	host       = "127.0.0.1"      // Target IP address
	portRange  = 65535            // Range of ports to scan
	maxWorkers = 100              // Number of concurrent workers
	outputFile = "open_ports.txt" // File to save open ports
	openPorts  []string           // Slice to store open ports
	mutex      sync.Mutex         // Mutex to protect shared resources
)

// checkPorts tests both TCP and UDP for a given port concurrently
func checkPorts(port int) (bool, bool) {
	var wg sync.WaitGroup
	var tcpOpen, udpOpen bool
	mutex := &sync.Mutex{}

	wg.Add(2)

	// Check TCP concurrently
	go func() {
		defer wg.Done()
		address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			mutex.Lock()
			tcpOpen = true
			mutex.Unlock()
			conn.Close()
		}
	}()

	// Check UDP concurrently
	go func() {
		defer wg.Done()
		address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		conn, err := net.DialTimeout("udp", address, 1*time.Second)
		if err == nil {
			mutex.Lock()
			udpOpen = true
			mutex.Unlock()
			conn.Close()
		}
	}()

	wg.Wait()
	return tcpOpen, udpOpen
}

// scanPort checks if a port is open for both TCP and UDP
func scanPort(wg *sync.WaitGroup, portChan chan int) {
	defer wg.Done()
	for port := range portChan {
		tcpOpen, udpOpen := checkPorts(port)
		var result string
		if tcpOpen {
			result = fmt.Sprintf("%s:%d (TCP)", host, port)
		}
		if udpOpen {
			if result != "" {
				result += " | "
			}
			result += fmt.Sprintf("%s:%d (UDP)", host, port)
		}
		if result != "" {
			fmt.Println(result)
			mutex.Lock()
			openPorts = append(openPorts, result)
			mutex.Unlock()
		}
	}
}

// saveResults writes the list of open ports to a file
func saveResults() {
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	for _, line := range openPorts {
		file.WriteString(line + "\n")
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
	close(portChan)

	wg.Wait()

	// Save results to file
	saveResults()
}
