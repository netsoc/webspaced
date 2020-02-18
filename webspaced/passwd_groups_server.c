#include <stdint.h>
#include <inttypes.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <pwd.h>
#include <sys/types.h>
#include <sys/select.h>
#include <sys/signalfd.h>
#include <sys/socket.h>
#include <sys/un.h>

#define DEBUG

#define REQ_GETPWNAM 0
#define REQ_GETGRNAM 1

#define RES_OK  0
#define RES_ERR 1

typedef struct req {
    int sock;
    uint8_t data[65536];
    ssize_t read;
    struct sockaddr_un src;
    socklen_t src_size;
} req_t;
void handle_req(req_t *r) {
    if (r->read != 5) {
        fprintf(stderr, "ignoring request with invalid length %zd from %s\n", read, r->src.sun_path);
        return;
    }

    uint8_t req_type = r->data[0];
    if (req_type == REQ_GETPWNAM) {
        uid_t uid = ((uid_t *)(r->data + 1))[0];
#ifdef DEBUG
        fprintf(stderr, "getting passwd entry for uid %" PRIu32 " on behalf of %s\n", uid, r->src.sun_path);
#endif
        struct passwd *p_entry = getpwuid(uid);
        if (p_entry == NULL) {
            perror("getpwuid()");
            r->data[0] = RES_ERR;
            if (sendto(r->sock, r->data, 1, 0, (struct sockaddr*)&r->src, r->src_size) == -1) {
                perror("sendto()");
            }
        }

#ifdef DEBUG
        fprintf(stderr, "uid %" PRIu32 " -> user \"%s\"\n", uid, p_entry->pw_name);
#endif
        r->data[0] = RES_OK;
        strcpy(r->data + 1, p_entry->pw_name);
        if (sendto(r->sock, r->data, 1 + strlen(p_entry->pw_name), 0, (struct sockaddr*)&r->src, r->src_size) == -1) {
            perror("sendto()");
        }
    } else {
        fprintf(stderr, "ignoring invalid request type '%" PRIu8 "' from %s\n", req_type, r->src.sun_path);
    }
}

int main(int argc, char **argv) {
    if (argc != 2) {
        fprintf(stderr, "usage: %s <unix socket>\n", argv[0]);
        return 1;
    }
    char *sock_path = argv[1];

    int sock = socket(AF_UNIX, SOCK_DGRAM, 0);
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
            req.sock = sock;
            req.src_size = sizeof(struct sockaddr_un);
            req.read = recvfrom(sock, req.data, sizeof(req.data), 0, (struct sockaddr*)&req.src, &req.src_size);

            if (req.read == -1) {
                perror("recvfrom()");
                ret = -5;
                goto error;
            } else if (req.read == 0) {
                break;
            }

            handle_req(&req);
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
