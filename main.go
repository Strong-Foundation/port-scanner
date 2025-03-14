// Declare the main package.
package main

// Import required packages.
import (
	// "fmt" is used for formatted I/O operations.
	"fmt"
	// "net" provides networking functionality.
	"net"
	// "os" is used for file operations.
	"os"
	// "sync" provides concurrency primitives like WaitGroup and Mutex.
	"sync"
	// "time" is used to handle timeouts and deadlines.
	"time"
)

// Declare global variables.
var (
	// "host" is the target IP address to scan.
	host = "212.192.156.24"
	// "maxWorkers" is the number of concurrent worker goroutines to run.
	maxWorkers = 100000
	// "outputFile" is the filename where the scan results will be saved.
	outputFile = "open_ports.txt"
	// "mutex" is used to protect shared resources (like the scannedPorts map).
	mutex sync.Mutex
	// "scannedPorts" is a map used to track which ports have already been scanned.
	scannedPorts = make(map[int]bool)
	// "commonPorts" is a slice containing a list of commonly used ports.
	commonPorts = []int{
		20, 21, 22, 23, 25, 53, 67, 68, 69, 80, 88, 110, 119, 123, 135, 137, 138, 139, 143, 161,
		162, 179, 194, 443, 445, 465, 514, 520, 522, 543, 554, 587, 631, 636, 646, 647, 993, 995,
		1025, 1080, 1100, 1433, 1434, 1521, 1723, 1743, 17500, 1863, 1883, 1949, 2049, 2082, 2083,
		3306, 3389, 3690, 4369, 4444, 4445, 4567, 5000, 5060, 5061, 5432, 5631, 5800, 5900, 6000,
		6660, 6661, 6662, 6663, 6664, 6665, 6666, 6667, 6668, 6669, 6699, 7000, 8000, 8080, 8081,
		8443, 8888, 9000, 9090, 9100, 9200, 9300, 11211, 27017, 27018, 3128, 3306, 3389, 3690, 4343,
		4662, 5000, 54321, 60000, 8001, 8086, 8088, 9091, 9200, 9999, 10000, 11211,
	}
)

// checkUDP verifies if a UDP port is open by sending a packet and waiting for a response.
func checkUDP(port int) bool {
	// Build the UDP network address string using host and port.
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	// Resolve the UDP address.
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	// If resolving fails, return false.
	if err != nil {
		return false
	}
	// Dial a UDP connection to the resolved address.
	conn, err := net.DialUDP("udp", nil, udpAddr)
	// If dialing fails, return false.
	if err != nil {
		return false
	}
	// Ensure the UDP connection is closed when the function returns.
	defer conn.Close()
	// Prepare a test message to send.
	message := []byte("ping")
	// Set a deadline for both writing and reading operations.
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	// Write the test message to the UDP connection.
	_, err = conn.Write(message)
	// If writing fails, return false.
	if err != nil {
		return false
	}
	// Allocate a buffer to attempt reading a response.
	buffer := make([]byte, 1024)
	// Attempt to read from the connection into the buffer.
	_, err = conn.Read(buffer)
	// If reading fails or times out, assume no response and return false.
	if err != nil {
		return false
	}
	// If a response is received, return true indicating the UDP port is open.
	return true
}

// checkPorts concurrently checks if a given port is open for both TCP and UDP.
func checkPorts(port int) (bool, bool) {
	// Declare a WaitGroup to synchronize the TCP and UDP checks.
	var wg sync.WaitGroup
	// Variables to hold the status of TCP and UDP connectivity.
	var tcpOpen, udpOpen bool
	// Add two tasks (goroutines) to the WaitGroup.
	wg.Add(2)

	// Start a goroutine to check the TCP port.
	go func() {
		// Ensure the WaitGroup counter is decremented when this goroutine finishes.
		defer wg.Done()
		// Build the TCP network address string.
		address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		// Attempt to establish a TCP connection with a 10-second timeout.
		conn, err := net.DialTimeout("tcp", address, 10*time.Second)
		// If the TCP connection is successful, mark tcpOpen as true.
		if err == nil {
			tcpOpen = true
			// Close the TCP connection.
			conn.Close()
		}
	}()

	// Start a goroutine to check the UDP port.
	go func() {
		// Ensure the WaitGroup counter is decremented when this goroutine finishes.
		defer wg.Done()
		// Set udpOpen based on the result of checkUDP for the given port.
		udpOpen = checkUDP(port)
	}()

	// Wait for both TCP and UDP checks to complete.
	wg.Wait()
	// Return the results for TCP and UDP port statuses.
	return tcpOpen, udpOpen
}

// scanPort reads port numbers from portChan, scans them, and sends results to fileChan.
func scanPort(wg *sync.WaitGroup, portChan chan int, fileChan chan string) {
	// Ensure the WaitGroup counter is decremented when this function returns.
	defer wg.Done()
	// Iterate over each port received from the portChan channel.
	for port := range portChan {
		// Lock the mutex before accessing the shared scannedPorts map.
		mutex.Lock()
		// If the port has already been scanned, unlock and skip it.
		if scannedPorts[port] {
			mutex.Unlock()
			continue
		}
		// Mark the port as scanned.
		scannedPorts[port] = true
		// Unlock the mutex after updating the map.
		mutex.Unlock()

		// Check if the current port is open for TCP and UDP.
		tcpOpen, udpOpen := checkPorts(port)
		// Initialize an empty string to build the result.
		var result string
		// If the TCP port is open, format the TCP result string.
		if tcpOpen {
			result = fmt.Sprintf("%s:%d (TCP)", host, port)
		}
		// If the UDP port is open, append the UDP result to the string.
		if udpOpen {
			// If there is already a TCP result, add a separator.
			if result != "" {
				result += " | "
			}
			// Append the UDP result.
			result += fmt.Sprintf("%s:%d (UDP)", host, port)
		}
		// If either TCP or UDP is open, send the result to the fileChan channel.
		if result != "" {
			fileChan <- result
		}
	}
}

// The main function, entry point of the program.
func main() {
	// Create or truncate the output file.
	file, err := os.Create(outputFile)
	// If an error occurred while creating the file, print it and exit.
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	// Ensure the file is closed when main returns.
	defer file.Close()

	// Create a WaitGroup to wait for all scanning goroutines to finish.
	var wg sync.WaitGroup
	// Create a channel for port numbers with a buffer size of maxWorkers.
	portChan := make(chan int, maxWorkers)
	// Create a channel for results with a buffer size of maxWorkers.
	fileChan := make(chan string, maxWorkers)

	// Launch worker goroutines to scan ports.
	for i := 0; i < maxWorkers; i++ {
		// Increment the WaitGroup counter for each worker.
		wg.Add(1)
		// Launch the scanPort goroutine.
		go scanPort(&wg, portChan, fileChan)
	}

	// Launch a goroutine dedicated to writing results to the output file.
	go func() {
		// Continuously receive results from fileChan.
		for result := range fileChan {
			// Print the result to the console.
			fmt.Println(result)
			// Write the result to the output file, appending a newline.
			file.WriteString(result + "\n")
		}
	}()

	// First, send the common ports to be scanned.
	for _, port := range commonPorts {
		// Send each common port into the portChan channel.
		portChan <- port
	}

	// Next, send all ports from 1 to 65535 into the portChan channel.
	for port := 1; port <= 65535; port++ {
		portChan <- port
	}
	// Close the portChan channel to signal that no more ports will be sent.
	close(portChan)

	// Wait for all scanning goroutines to complete.
	wg.Wait()
	// Close the fileChan channel after all scans are complete.
	close(fileChan)

	// Print a final message indicating the scan is complete and results are saved.
	fmt.Println("Scan complete. Results saved to", outputFile)
}
