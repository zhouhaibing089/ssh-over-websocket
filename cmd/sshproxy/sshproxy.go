package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var (
	username   string
	password   string
	sshKeyFile string

	bindAddress string
	port        int
	tlsCertFile string
	tlsKeyFile  string

	sshConfig *ssh.ClientConfig
)

func init() {
	// flags related to ssh config
	flag.StringVar(&username, "user", "", "username for ssh login")
	flag.StringVar(&password, "password", "", "password paired with username for ssh login")
	flag.StringVar(&sshKeyFile, "ssh-key-file", "", "path to ssh key paired with username for ssh login")

	// flags related to server config.
	flag.StringVar(&bindAddress, "bind-address", "", "the listenning address of this server")
	flag.IntVar(&port, "port", 0, "the listenning port of this server")
	flag.StringVar(&tlsCertFile, "tls-cert-file", "", "path to tls certificate for https serving")
	flag.StringVar(&tlsKeyFile, "tls-key-file", "", "path to tls key for https serving")
}

func main() {
	flag.Parse()

	// password and sshKeyFile can not be set at the same time
	if password != "" && sshKeyFile != "" {
		log.Fatalf("--password and --ssh-key-file can not be used together")
	}
	sshConfig = &ssh.ClientConfig{
		User: username,
	}
	if password != "" {
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.Password(password),
		}
	}
	if sshKeyFile != "" {
		keyBytes, err := ioutil.ReadFile(sshKeyFile)
		if err != nil {
			log.Fatalf("failed to read key: %s", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			log.Fatalf("failed to parse key: %s", err)
		}
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}
	// skip host key check
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	router := mux.NewRouter()
	router.HandleFunc("/ssh/{host}", handleSSH)

	var err error
	if tlsCertFile != "" && tlsKeyFile != "" {
		err = http.ListenAndServeTLS(
			fmt.Sprintf("%s:%d", bindAddress, port),
			tlsCertFile, tlsKeyFile, router,
		)
	} else {
		err = http.ListenAndServe(
			fmt.Sprintf("%s:%d", bindAddress, port),
			router,
		)
	}
	if err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}

func handleSSH(w http.ResponseWriter, r *http.Request) {
	wstr := r.URL.Query().Get("width")
	hstr := r.URL.Query().Get("height")
	width, werr := strconv.ParseInt(wstr, 10, 32)
	height, herr := strconv.ParseInt(hstr, 10, 32)
	if werr != nil || herr != nil {
		log.Printf("width or height not given: %s, %s", werr, herr)
		return
	}

	host := mux.Vars(r)["host"]
	client, err := ssh.Dial("tcp", host+":22", sshConfig)
	if err != nil {
		log.Printf("failed to dial %s: %s", host, err)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Printf("failed to new session: %s", err)
		return
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		log.Printf("failed to get stdin pipe: %s", err)
		return
	}
	defer stdin.Close()
	stdout, err := session.StdoutPipe()
	if err != nil {
		log.Printf("failed to stdout pipe: %s", err)
		return
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("linux", int(height), int(width), modes); err != nil {
		log.Printf("failed to request for pseudo terminal: %s", err)
		return
	}
	if err := session.Shell(); err != nil {
		log.Printf("failed to start shell: %s", err)
		return
	}

	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		log.Printf("failed to do protocol upgrade: %s", err)
		return
	}
	defer conn.Close()

	closed := false

	// read from client
	go func() {
		for {
			if closed {
				return
			}

			mt, reader, err := conn.NextReader()
			if err != nil {
				log.Printf("failed to read from client: %s", err)
				return
			}
			if mt == websocket.TextMessage {
				sizeBytes, err := ioutil.ReadAll(reader)
				if err != nil {
					log.Printf("failed to read text message from client: %s", err)
				}

				sizes := strings.Split(string(sizeBytes), ",")
				if len(sizes) != 2 {
					continue
				}
				width, werr := strconv.ParseInt(sizes[0], 10, 32)
				height, herr := strconv.ParseInt(sizes[1], 10, 32)
				if werr != nil || herr != nil {
					continue
				}

				err = session.WindowChange(int(height), int(width))
				if err != nil {
					log.Printf("failed to change window: %s", err)
				}
			}
			// ignore non binary message.
			if mt != websocket.BinaryMessage {
				continue
			}
			_, err = io.Copy(stdin, reader)
			if err != nil {
				log.Printf("failed to write into remote: %s", err)
				return
			}
		}
	}()

	// read from ssh server
	for {
		buf := make([]byte, 1024)
		n, err := stdout.Read(buf)
		if err != nil {
			if err == io.EOF {
				closed = true
				return
			}
			log.Printf("failed to read from remote: %s", err)
			return
		}
		err = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
		if err != nil {
			log.Printf("failed to write into client: %s", err)
			return
		}
	}
}
