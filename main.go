package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

// Telnet Protocol Constants
const (
	IAC  = 255 // Interpret As Command
	DONT = 254
	DO   = 253
	WONT = 252
	WILL = 251
	SB   = 250 // Subnegotiation Begin
	SE   = 240 // Subnegotiation End
)

// Config holds the runtime configuration
type Config struct {
	Host    string
	Port    string
	LogFile string
}

func main() {
	// 1. Parse command-line arguments (matching standard telnet behavior)
	config := parseArgs()

	// 2. Connect to the target server
	target := net.JoinHostPort(config.Host, config.Port)
	fmt.Printf("[*] Connecting to %s...\r\n", target)
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Fatalf("[-] Connection failed: %v", err)
	}
	defer conn.Close()
	fmt.Printf("[+] Connected! Use Ctrl+C to exit.\r\n")

	// 3. Set the local terminal to Raw Mode (Crucial step)
	// This allows us to capture keystrokes byte-by-byte instantly
	// and prevents the local shell from intercepting signals like Ctrl+C.
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatalf("[-] Failed to set raw mode: %v", err)
	}
	// Ensure the terminal state is restored upon program exit,
	// otherwise the terminal will be left in a broken state.
	defer term.Restore(fd, oldState)

	// 4. Prepare output stream (Support optional logging)
	var outputWriter io.Writer = os.Stdout
	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Careful with printing errors in Raw Mode (needs \r\n)
			fmt.Fprintf(os.Stderr, "[-] Failed to open log file: %v\r\n", err)
		} else {
			defer f.Close()
			// Write to both Stdout and the log file simultaneously
			outputWriter = io.MultiWriter(os.Stdout, f)
			fmt.Fprintf(os.Stdout, "[+] Logging session to: %s\r\n", config.LogFile)
		}
	}

	// 5. Handle system signals (for graceful shutdown)
	handleSignals(conn)

	// 6. Start full-duplex communication channels
	errChan := make(chan error, 1)

	// Goroutine A: Network -> Screen/File (Handles Telnet protocol filtering)
	go func() {
		// Use a custom Telnet Reader to strip/handle IAC commands
		telnetReader := NewTelnetReader(conn)
		_, err := io.Copy(outputWriter, telnetReader)
		errChan <- err
	}()

	// Goroutine B: Keyboard -> Network
	go func() {
		// Forward keyboard input directly to the socket
		_, err := io.Copy(conn, os.Stdin)
		errChan <- err
	}()

	// Wait for either goroutine to finish (e.g., connection lost or user exit)
	<-errChan
	fmt.Printf("\r\n[*] Connection closed by foreign host.\r\n")
}

// parseArgs parses arguments to match standard telnet: "telnet <host> [port]"
func parseArgs() Config {
	logFile := flag.String("log", "", "Log output to file (optional)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-log filename] <host> [port]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	host := args[0]
	port := "23" // Default telnet port

	if len(args) >= 2 {
		port = args[1]
	}

	return Config{
		Host:    host,
		Port:    port,
		LogFile: *logFile,
	}
}

// handleSignals captures Ctrl+C or Termination signals
func handleSignals(conn net.Conn) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		conn.Close()
		// Restoration of the terminal state is handled by the defer in main()
		os.Exit(0)
	}()
}

// ==========================================
// Telnet Protocol Handler (Core Logic)
// ==========================================

// TelnetReader wraps net.Conn to filter Telnet commands
type TelnetReader struct {
	reader *bufio.Reader
}

func NewTelnetReader(r io.Reader) *TelnetReader {
	return &TelnetReader{
		reader: bufio.NewReader(r),
	}
}

// Read implements the io.Reader interface.
// It acts as a simplified state machine to strip IAC commands,
// returning only the pure text data.
func (t *TelnetReader) Read(p []byte) (n int, err error) {
	// Read byte by byte to handle the state machine correctly.
	for n < len(p) {
		b, err := t.reader.ReadByte()
		if err != nil {
			return n, err
		}

		// If IAC (Command start) is encountered
		if b == IAC {
			// Read the next byte to see what the command is
			cmd, err := t.reader.ReadByte()
			if err != nil {
				return n, err
			}

			if cmd == IAC {
				// 0xFF 0xFF means literal 0xFF data
				p[n] = IAC
				n++
			} else if cmd == DO || cmd == DONT || cmd == WILL || cmd == WONT {
				// Negotiation command: IAC [DO/DONT/WILL/WONT] [Option]
				// We simply ignore the option byte (refusing negotiation, remaining Dumb)
				_, err = t.reader.ReadByte()
				if err != nil {
					return n, err
				}
			} else if cmd == SB {
				// Subnegotiation: IAC SB ... IAC SE
				// Loop until IAC SE is encountered
				for {
					sbBytes, err := t.reader.ReadByte()
					if err != nil {
						return n, err
					}
					if sbBytes == IAC {
						next, err := t.reader.ReadByte()
						if err != nil {
							return n, err
						}
						if next == SE {
							break // End of subnegotiation
						}
					}
				}
			} else {
				// Other commands (NOP, DM, BRK, IP...) are simply ignored
				continue
			}
		} else {
			// Regular text data
			p[n] = b
			n++
		}

		// If the buffer has some data and the network stream has paused,
		// return immediately to ensure smooth rendering on the terminal.
		if t.reader.Buffered() == 0 && n > 0 {
			break
		}
	}
	return n, nil
}
