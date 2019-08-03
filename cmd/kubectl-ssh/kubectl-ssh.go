package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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
	go func() {
		for {
			mt, r, err := conn.NextReader()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			if err != nil {
				terminal.Restore(inFD, state)
				cancel()
				return
			}
			if mt != websocket.BinaryMessage {
				terminal.Restore(inFD, state)
				cancel()
				return
			}
			if _, err := io.Copy(os.Stdout, r); err != nil {
				log.Printf("failed to read from server: %v", err)
				terminal.Restore(inFD, state)
				cancel()
			}
		}
	}()
	if err := File2WS(ctx, cancel, os.Stdin, conn); err == io.EOF {
		if err := conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(10*time.Second)); err == websocket.ErrCloseSent {
		} else if err != nil {
			log.Printf("failed to send close message: %v", err)
		}
	} else if err == websocket.ErrCloseSent {
		terminal.Restore(inFD, state)
		cancel()
	}
}

func File2WS(ctx context.Context, cancel func(), src io.Reader, dst *websocket.Conn) error {
	defer cancel()
	for {
		if ctx.Err() != nil {
			return nil
		}
		b := make([]byte, 32*1024)
		if n, err := src.Read(b); err != nil {
			return err
		} else {
			b = b[:n]
		}
		if err := dst.WriteMessage(websocket.BinaryMessage, b); err != nil {
			return err
		}
	}
}
