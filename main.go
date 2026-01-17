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

// ANSI Escape Sequences for terminal control
const (
	AnsiClearScreen = "\033[H\033[2J" // Move cursor home and clear screen
	AnsiSetTitle    = "\033]0;%s\007" // Set window/tab title
)

// Config holds the runtime configuration
type Config struct {
	Host    string
	Port    string
	LogFile string
}

func main() {
	// 1. Parse command-line arguments
	config := parseArgs()

	// 2. Connect to the target server
	target := net.JoinHostPort(config.Host, config.Port)
	fmt.Printf("[*] Connecting to %s...\r\n", target)

	// Set a connection timeout
	conn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Fatalf("[-] Connection failed: %v", err)
	}
	defer conn.Close()

	// 3. Set the local terminal to Raw Mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		log.Fatalf("[-] Failed to set raw mode: %v", err)
	}
	// Ensure terminal state is restored on exit
	defer term.Restore(fd, oldState)

	// 4. IMPROVEMENT: Initialize terminal view (Clear screen & Set Title)
	// We do this AFTER setting Raw Mode to ensure full control over output
	setupTerminalOutput(config.Host, config.Port)

	// 5. Prepare output stream (Support optional logging)
	var outputWriter io.Writer = os.Stdout
	if config.LogFile != "" {
		f, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[-] Failed to open log file: %v\r\n", err)
		} else {
			defer f.Close()
			outputWriter = io.MultiWriter(os.Stdout, f)
			// Print a session start marker to the log/screen
			fmt.Fprintf(outputWriter, "--- Session Start: %s ---\r\n", time.Now().Format(time.RFC3339))
		}
	}

	// 6. Handle system signals
	handleSignals(conn)

	// 7. Start full-duplex communication channels
	errChan := make(chan error, 1)

	// Goroutine A: Network -> Screen/File
	go func() {
		telnetReader := NewTelnetReader(conn)
		_, err := io.Copy(outputWriter, telnetReader)
		errChan <- err
	}()

	// Goroutine B: Keyboard -> Network
	go func() {
		_, err := io.Copy(conn, os.Stdin)
		errChan <- err
	}()

	// Wait for exit
	<-errChan
	fmt.Printf("\r\n[*] Connection closed by foreign host.\r\n")
}

// setupTerminalOutput handles visual improvements like clearing screen and setting tab title
func setupTerminalOutput(host, port string) {
	// 1. Clear the screen so output starts from the top
	fmt.Print(AnsiClearScreen)

	// 2. Set the Windows Terminal Tab Title to "Telnet host:port"
	title := fmt.Sprintf("Telnet %s:%s", host, port)
	fmt.Printf(AnsiSetTitle, title)

	// 3. Print a friendly banner at the very top
	fmt.Printf("Connected to %s:%s\r\n", host, port)
	fmt.Printf("Use Ctrl+C to exit.\r\n")
	fmt.Printf("----------------------------------------------------------------\r\n")
}

// parseArgs parses arguments
func parseArgs() Config {
	logFile := flag.String("log", "", "Log output to file (optional)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-log filename] <host> [port]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	host := args[0]
	port := "23"

	if len(args) >= 2 {
		port = args[1]
	}

	return Config{
		Host:    host,
		Port:    port,
		LogFile: *logFile,
	}
}

// handleSignals captures Ctrl+C
func handleSignals(conn net.Conn) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		conn.Close()
		os.Exit(0)
	}()
}

// Telnet Protocol Handler
type TelnetReader struct {
	reader *bufio.Reader
}

func NewTelnetReader(r io.Reader) *TelnetReader {
	return &TelnetReader{
		reader: bufio.NewReader(r),
	}
}

func (t *TelnetReader) Read(p []byte) (n int, err error) {
	for n < len(p) {
		b, err := t.reader.ReadByte()
		if err != nil {
			return n, err
		}

		if b == IAC {
			cmd, err := t.reader.ReadByte()
			if err != nil {
				return n, err
			}

			if cmd == IAC {
				p[n] = IAC
				n++
			} else if cmd == DO || cmd == DONT || cmd == WILL || cmd == WONT {
				_, err = t.reader.ReadByte()
				if err != nil {
					return n, err
				}
			} else if cmd == SB {
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
							break
						}
					}
				}
			} else {
				continue
			}
		} else {
			p[n] = b
			n++
		}

		if t.reader.Buffered() == 0 && n > 0 {
			break
		}
	}
	return n, nil
}
