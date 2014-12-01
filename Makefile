all:
	go build example.go server.go
	go build example_buffered.go server.go buffered_work.go 
	go build example_batched.go server.go maximally_batched_work.go 
	gcc -o forkexec -std=c99 example.c gochannel.c
	gcc -o example_batched_exec -std=c99 example_batched.c gochannel.c

test:
	./forkexec ./example_buffered
	./forkexec ./example
	echo "hello" | ./example_buffered
	./example_batched_exec ./example_batched
clean:
	rm forkexec example example_buffered example_batched example_batched_exec