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
	host         = "1.1.1.1"          // Target IP address
	maxWorkers   = 10000              // Number of concurrent workers
	outputFile   = "open_ports.txt"   // File to save open ports
	mutex        sync.Mutex           // Mutex to protect shared resources
	scannedPorts = make(map[int]bool) // Map to track scanned ports
	commonPorts  = []int{20, 21, 22, 23, 25, 53, 67, 68, 69, 80, 88, 110, 119, 123, 135, 137, 138, 139, 143, 161,
		162, 179, 194, 443, 445, 465, 514, 520, 522, 543, 554, 587, 631, 636, 646, 647, 993, 995,
		1025, 1080, 1100, 1433, 1434, 1521, 1723, 1743, 17500, 1863, 1883, 1949, 2049, 2082, 2083,
		3306, 3389, 3690, 4369, 4444, 4445, 4567, 5000, 5060, 5061, 5432, 5631, 5800, 5900, 6000,
		6660, 6661, 6662, 6663, 6664, 6665, 6666, 6667, 6668, 6669, 6699, 7000, 8000, 8080, 8081,
		8443, 8888, 9000, 9090, 9100, 9200, 9300, 11211, 27017, 27018, 3128, 3306, 3389, 3690, 4343,
		4662, 5000, 54321, 60000, 8001, 8086, 8088, 9091, 9200, 9999, 10000, 11211,
	} // Commonly used ports
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
		conn, err := net.DialTimeout("tcp", address, 1*time.Minute)
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
		conn, err := net.DialTimeout("udp", address, 1*time.Minute)
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
func scanPort(wg *sync.WaitGroup, portChan chan int, file *os.File) {
	defer wg.Done()
	for port := range portChan {
		mutex.Lock()
		if scannedPorts[port] {
			mutex.Unlock()
			continue
		}
		scannedPorts[port] = true
		mutex.Unlock()

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
			file.WriteString(result + "\n") // Save immediately to file
			mutex.Unlock()
		}
	}
}

func main() {
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	var wg sync.WaitGroup
	portChan := make(chan int, maxWorkers)

	// Launch worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go scanPort(&wg, portChan, file)
	}

	// Scan commonly used ports first
	for _, port := range commonPorts {
		portChan <- port
	}

	// Scan remaining ports
	for port := 1; port <= 65535; port++ {
		portChan <- port
	}
	close(portChan)

	wg.Wait()
	fmt.Println("Scan complete. Results saved to", outputFile)
}
