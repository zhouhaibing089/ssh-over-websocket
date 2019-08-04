package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <host>\n", os.Args[0])
		os.Exit(1)
	}
	host := os.Args[1]
	inFD := int(os.Stdin.Fd())
	state, err := terminal.MakeRaw(inFD)
	if err != nil {
		log.Printf("failed to make raw: %s\n", err)
		return
	}
	defer terminal.Restore(inFD, state)
	width, height, err := terminal.GetSize(inFD)
	if err != nil {
		log.Printf("failed to get terminal size: %s", err)
		return
	}

	dialer := websocket.Dialer{}
	urlStr := fmt.Sprintf("ws://localhost:8080/ssh/%s?width=%d&height=%d", host, width, height)
	conn, _, err := dialer.Dial(urlStr, nil)
	if err != nil {
		log.Printf("failed to dial: %s\n", err)
		return
	}
	defer conn.Close()
	stopCh := make(chan struct{}, 1)
	defer func() {
		stopCh <- struct{}{}
		close(stopCh)
	}()
	// watching for terminal size event
	go func() {
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, unix.SIGWINCH)
		defer signal.Stop(winch)
		for {
			select {
			case <-winch:
				width, height, err := terminal.GetSize(inFD)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to get terminal size: %s", err)
					return
				}
				// update the message to remote
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("%d,%d", width, height)))
			case <-stopCh:
				return
			}
		}
	}()

	// send the output from server to stdout
	closed := false
	go func() {
		for {
			mt, r, err := conn.NextReader()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				closed = true
				return
			}
			if err != nil {
				log.Printf("failed to get next reader: %s", err)
				return
			}
			if mt != websocket.BinaryMessage {
				continue
			}
			if _, err := io.Copy(os.Stdout, r); err != nil {
				log.Printf("failed to read from server: %v", err)
				return
			}
		}
	}()

	for {
		buf := make([]byte, 32*1024)
		if n, err := os.Stdin.Read(buf); err != nil {
			log.Printf("failed to read from stdin: %s", err)
			return
		} else {
			buf = buf[:n]
		}
		if closed {
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
			log.Printf("failed to write to server: %s", err)
			return
		}
	}
}
