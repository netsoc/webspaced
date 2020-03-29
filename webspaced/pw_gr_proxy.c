#include <stdbool.h>
#include <stdint.h>
#include <inttypes.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <pwd.h>
#include <grp.h>
#include <sys/types.h>
#include <sys/select.h>
#include <sys/signalfd.h>
#include <sys/socket.h>
#include <sys/un.h>

#define REQ_GETPWUID        0
#define REQ_USER_IS_MEMBER  1

#define RES_OK  0
#define RES_ERR 1

void send_error(int sock) {
    uint8_t data = RES_ERR;
    if (send(sock, &data, 1, 0) == -1) {
        perror("sendto()");
    }
}

typedef struct req {
    int sock;
    struct sockaddr_un src;
    socklen_t src_size;
} req_t;
void handle_req(req_t *r) {
    uint8_t data[65536];
    ssize_t n = recv(r->sock, data, sizeof(data), 0);
    if (n == -1) {
        perror("recv()");
        return;
    } else if (n == 0) {
        fprintf(stderr, "\"%s\" closed connection unexpectedly", r->src.sun_path);
        return;
    }


    uint8_t req_type = data[0];
    if (req_type == REQ_GETPWUID) {
        if (n != 5) {
            fprintf(stderr, "ignoring getpwuid request with invalid length %zd from \"%s\"\n", n, r->src.sun_path);
            return;
        }

        uid_t uid = ((uid_t *)(data + 1))[0];
#ifdef DEBUG
        fprintf(stderr, "getting passwd entry for uid %" PRIu32 " on behalf of \"%s\"\n", uid, r->src.sun_path);
#endif
        struct passwd *p_entry = getpwuid(uid);
        if (p_entry == NULL) {
            perror("getpwuid()");
            send_error(r->sock);
            return;
        }

#ifdef DEBUG
        fprintf(stderr, "uid %" PRIu32 " -> user \"%s\"\n", uid, p_entry->pw_name);
#endif
        data[0] = RES_OK;
        strcpy(data + 1, p_entry->pw_name);
        if (send(r->sock, data, 1 + strlen(p_entry->pw_name), 0) == -1) {
            perror("send()");
        }
    } else if (req_type == REQ_USER_IS_MEMBER) {
        char *username = (char*)(data + 1);
        char *group = username + strlen(username) + 1;
#ifdef DEBUG
        fprintf(stderr, "checking if \"%s\" is a member of \"%s\" on behalf of \"%s\"\n", username, group, r->src.sun_path);
#endif

        struct group *gr = getgrnam(group);
        if (gr == NULL) {
            perror("getgrnam()");
            send_error(r->sock);
            return;
        }

        bool is_mem = false;
        for (char **p = gr->gr_mem; *p != NULL; p++) {
            if (strcmp(*p, username) == 0) {
                is_mem = true;
                break;
            }
        }

        data[0] = RES_OK;
        data[1] = is_mem;
        if (send(r->sock, data, 2, 0) == -1) {
            perror("send()");
        }
    } else {
        fprintf(stderr, "ignoring invalid request type '%" PRIu8 "' from \"%s\"\n", req_type, r->src.sun_path);
    }
}

int main(int argc, char **argv) {
    if (argc != 2) {
        fprintf(stderr, "usage: %s <unix socket>\n", argv[0]);
        return 1;
    }
    char *sock_path = argv[1];

    int sock = socket(AF_UNIX, SOCK_SEQPACKET, 0);
    if (sock == -1) {
        perror("socket()");
        return -1;
    }

    struct sockaddr_un bind_addr = {0};
    bind_addr.sun_family = AF_UNIX;
    strncpy(bind_addr.sun_path, sock_path, sizeof(bind_addr.sun_path) - 1);

    int ret = 0;
    if (bind(sock, (struct sockaddr*)&bind_addr, sizeof(struct sockaddr_un)) == -1) {
        perror("bind()");
        ret = -2;
        goto error_psfd;
    }

    if (listen(sock, 16) == -1) {
        perror("listen()");
        ret = -2;
        goto error_psfd;
    }

    sigset_t mask;
    sigemptyset(&mask);
    sigaddset(&mask, SIGINT);
    sigaddset(&mask, SIGTERM);
    if (sigprocmask(SIG_BLOCK, &mask, NULL) == -1) {
        perror("sigprocmask()");
        ret = -3;
        goto error_psfd;
    }

    int sfd = signalfd(-1, &mask, 0);
    if (sfd == -1) {
        perror("signalfd()");
        ret = -3;
        goto error_psfd;
    }

    int nfds = (sfd > sock ? sfd : sock) + 1;
    fd_set rfds;
    for (;;) {
        FD_ZERO(&rfds);
        FD_SET(sfd, &rfds);
        FD_SET(sock, &rfds);
        int ret = select(nfds, &rfds, NULL, NULL, NULL);
        if (ret == -1) {
            perror("select()");
            ret = -4;
            goto error;
        } else if (ret == 0) {
            continue;
        }

        if (FD_ISSET(sock, &rfds)) {
            req_t req = {0};
            req.src_size = sizeof(struct sockaddr_un);
            req.sock = accept(sock, (struct sockaddr*)&req.src, &req.src_size);

            if (req.sock == -1) {
                perror("accept()");
                ret = -5;
                goto error;
            }

            handle_req(&req);
            close(req.sock);
        }
        if (FD_ISSET(sfd, &rfds)) {
            break;
        }
    }

    fprintf(stderr, "shutting down\n");
error:
    close(sfd);
error_psfd:
    close(sock);
    unlink(sock_path);
    return ret;
}
