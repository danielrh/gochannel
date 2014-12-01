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
	"io"
	"log"
)

// this reads in a loop from socketRead putting batchSize bytes of work to copyTo until
// the socketRead is empty. Will always block until a full workSize of units have been copied
func readBuffer(copyTo chan<- []byte, socketRead io.Reader, batchSize int, workSize int) {
	defer close(copyTo)
	for {
		batch := make([]byte, batchSize)
		size, err := socketRead.Read(batch)
		if err == nil && size%workSize != 0 {
			var lsize int
			lsize, err = io.ReadFull(socketRead, batch[size:size+workSize-(size%workSize)])
			size += lsize
		}
		if size > 0 {
			copyTo <- batch[:size]
		}
		if err != nil {
			if err != io.EOF {
				log.Print("Error encountered in readBuffer:", err)
			}
			return
		}
	}
}

// this simply copies data from the chan to the socketWrite writer
func writeBuffer(copyFrom <-chan []byte, socketWrite io.Writer) {
	for buf := range copyFrom {
		if len(buf) > 0 {
			_, err := socketWrite.Write(buf)
			if err != nil {
				log.Print("Error encountered in writeBuffer:", err)
				return
			}
		}
	}
}

// this function takes data from socketRead and calls processBatch on a batch of it at a time
// then the resulting bytes are written to wocketWrite as fast as possible
func processBufferedData(socketRead io.Reader, socketWrite io.Writer,
	makeProcessBatch func() (func(input []byte) []byte,
		func(lastInput []byte, lastOutput []byte)),
	batchSize, workItemSize int) {
	readChan := make(chan []byte, 2)
	writeChan := make(chan []byte, 1+batchSize/workItemSize)
	go readBuffer(readChan, socketRead, batchSize, workItemSize)
	go writeBuffer(writeChan, socketWrite)
	pastInit := false
	defer func() { // this is if makeProcessBatch() fails
		if !pastInit {
			if r := recover(); r != nil {
				log.Print("Error in makeProcessBatch ", r)
			}
		}
		close(writeChan)
	}()
	processBatch, prefetchBatch := makeProcessBatch()
	pastInit = true
	for buf := range readChan {
		result := processBatch(buf)
		writeChan <- result
		prefetchBatch(buf, result)
	}
}
