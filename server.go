/////////////////////////////////////////////////////////////////////////////////////////////////
// Copyright (c) 2014, Daniel Reiter Horn
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification, are permitted
// provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this list of
//    conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice, this list of
//    conditions and the following disclaimer in the documentation and/or other materials
//    provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR
// IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY
// AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
// CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR
// OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.
//////////////////////////////////////////////////////////////////////////////////////////////////

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log"
	"net"
	"os"
)

func messageOnCloseAndRun(exitChan chan<- bool,
	socketRead io.Reader, socketWrite io.Writer, process func(io.Reader, io.Writer)) {
	process(socketRead, socketWrite)
	exitChan <- true
}

func validateAndRun(token []byte,
	socketRead io.Reader, socketWrite io.Writer, process func(io.Reader, io.Writer)) {
	test := make([]byte, len(token))
	_, token_err := io.ReadFull(socketRead, test[:])
	if token_err == nil && bytes.Equal(token, test[:]) {
		process(socketRead, socketWrite)
	} else {
		log.Print("Error: token mismatch from new client")
	}
}

func listenAccept(newConnection chan<- net.Conn, l net.Listener) {
	for {
		fd, err := l.Accept()
		if err != nil {
			log.Print("accept error:", err)
		} else {
			newConnection <- fd
		}
	}
}

func startServer(process func(io.Reader, io.Writer)) {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	hexToken := make([]byte, 32)
	{
		token := make([]byte, 16)
		rand.Read(token)
		hex.Encode(hexToken, token)
	}
	filePathReturn := []byte("/tmp/go-" + base64.URLEncoding.EncodeToString(uuid))
	if len(filePathReturn) != 32 {
		log.Fatal("File path is not 32 bytes " + string(filePathReturn))
	}
	filePathReturn[31] = '\n' // newline instead of padding with =
	filePath := string(filePathReturn[:31])

	l, err := net.Listen("unix", string(filePath))
	if err != nil {
		log.Print("listen error:", err)
	}
	defer os.Remove(filePath)
	pathAndToken := string(filePathReturn) + string(hexToken)
	_, err = os.Stdout.Write([]byte(pathAndToken))
	if err != nil {
		panic(err)
	}
	exitChan := make(chan bool)
	connectionChan := make(chan net.Conn)
	go messageOnCloseAndRun(exitChan, os.Stdin, os.Stdout, process)
	go listenAccept(connectionChan, l)
	for {
		select { // FIXME: make this loop terminate if stdin or stdout close
		case <-exitChan:
			return
		case fd := <-connectionChan:
			go validateAndRun(hexToken, fd, fd, process)
		}
	}
}
