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

#include <stddef.h>
#include <stdlib.h>
#include <stdio.h>
#include <assert.h>
#include <unistd.h>
#include <string.h>
#include <sys/socket.h>
#include <sys/un.h>
#include "gochannel.h"

ptrdiff_t read_until(int fd, void *buf, int size) {
    size_t progress = 0;
    while (progress < size) {
        ptrdiff_t status = read(fd, (char*)buf + progress, size - progress);
        if (status == -1) {
            return -1;
        }
        progress += status;
    }
    return progress;
}

ptrdiff_t write_until(int fd, void *buf, int size) {
    size_t progress = 0;
    while (progress < size) {
        ptrdiff_t status = write(fd, (char*)buf + progress, size - progress);
        if (status == -1) {
            return -1;
        }
        progress += status;
    }
    return progress;
}


// static assert that we have sufficient room in our struct
static char static_assert[sizeof((struct sockaddr_un*)0)->sun_path - GO_CHANNEL_PATH_LENGTH];

struct GoChannel clone_go_channel(struct GoChannel parent) {
    parent.stdout = -1;
    parent.stdin = -1;

    struct sockaddr_un address;
    memset(&address, 0, sizeof(struct sockaddr_un));
    int socket_fd;
    socket_fd = socket(PF_UNIX, SOCK_STREAM, 0);
    if(socket_fd < 0) {
        return parent;
    }
    address.sun_family = AF_UNIX;
    memcpy(address.sun_path, parent.path, sizeof(parent.path));
    if (connect(socket_fd, (struct sockaddr *)&address, sizeof(struct sockaddr_un)) != 0) {
        return parent;
    }
    write_until(socket_fd, parent.token, sizeof(parent.token));
    parent.stdout = socket_fd;
    parent.stdin = socket_fd;
    return parent;
}
struct GoChannel launch_go_subprocess(const char* path_to_exe, char *const argv[]) {
    struct GoChannel ret;
    int subprocess_stdin[2];
    int subprocess_stdout[2];
    pipe(subprocess_stdin);
    pipe(subprocess_stdout);
    ret.stdin = subprocess_stdin[1];
    ret.stdout = subprocess_stdout[0];
    if (fork() == 0) {
        fclose(stdin);
        dup(subprocess_stdin[0]);
        close(subprocess_stdin[1]);
        fclose(stdout);
        dup(subprocess_stdout[1]);
        close(subprocess_stdout[0]);
        execvp(path_to_exe, argv);
    }else {
        close(subprocess_stdin[0]);
        close(subprocess_stdout[1]);
        int path_status = read_until(ret.stdout, (unsigned char*)ret.path, sizeof(ret.path));
        ret.path[sizeof(ret.path) - 1] = '\0';
        int token_status = read_until(ret.stdout, ret.token, sizeof(ret.token));
        assert(path_status != -1);
        assert(token_status != -1);
    }
    return ret;
}

void close_go_channel(struct GoChannel *channel) {
    if (channel->stdout != -1) {
        close(channel->stdout);
    }
    if (channel->stdin != -1) {
        close(channel->stdin);
    }
    channel->stdout = -1;
    channel->stdin = -1;
}
