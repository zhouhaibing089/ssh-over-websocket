sshproxy
========

sshproxy is an http server which proxies interactions to an ssh session.

## Install

```console
$ go get github.com/zhouhaibing089/sshproxy/cmd/sshproxy
```

## Usage

The sshproxy accepts two kinds of ssh authentication:

```console
$ sshproxy --user=<ssh user> \
    --key=<ssh key> \
    --bind-address=127.0.0.1 \
    --port=8443 \
    --tls-cert-file=<tls certificate> \
    --tls-private-key-file=<tls private key file>
```