/*
 * Synthetic matchmaking example; no game assets or peer transport required.
 * From the repository root:
 *   cc -std=gnu11 -Wall -Wextra -Iclient client/psnr.c client/examples/minimal.c -o minimal
 * Run: ./minimal HOST PORT NAME
 *
 * psnr_connect and psnr_call block: this is a setup demo, not a frame loop.
 * An asynchronous integration gives one owner the client, uses psnr_send,
 * pumps psnr_poll regularly, correlates replies by request ID, and frees every
 * returned message. See docs/api.md for payloads and limits.
 */
#include "psnr.h"

#include <inttypes.h>
#include <stdio.h>
#include <string.h>

static int parse_port(const char* text, uint16_t* out)
{
    unsigned port = 0;
    if (!*text) return 0;
    for (; *text; text++) {
        if (*text < '0' || *text > '9') return 0;
        port = port * 10 + (unsigned)(*text - '0');
        if (port > 65535) return 0;
    }
    if (!port) return 0;
    *out = (uint16_t)port;
    return 1;
}

/* Validate lengths before reading any multi-byte field, including blobs in
 * entries we do not display. ROOM_LIST has a 4-byte header and 32-byte entries
 * followed by their variable-length external data. */
static int room_list_valid(const psnr_msg* msg)
{
    size_t off = 4;
    if (msg->len < 4) return 0;
    uint16_t total = psnr_get16(msg->data);
    uint16_t count = psnr_get16(msg->data + 2);
    if (count > 10 || count > total) return 0;
    for (uint16_t i = 0; i < count; i++) {
        if (msg->len - off < 32) return 0;
        uint16_t blob_len = psnr_get16(msg->data + off + 30);
        off += 32;
        if (blob_len > msg->len - off) return 0;
        off += blob_len;
    }
    return off == msg->len;
}

int main(int argc, char** argv)
{
    uint16_t port;
    uint32_t user_id;
    uint8_t public_ip[4], body[4];
    psnr_msg reply;
    int result = 1;

    if (argc != 4) {
        fprintf(stderr, "Usage: %s HOST PORT NAME\n", argv[0]);
        return 2;
    }
    if (!parse_port(argv[2], &port)) {
        fprintf(stderr, "PORT must contain decimal digits in the range 1..65535\n");
        return 2;
    }
    if (!*argv[1] || !*argv[3] || strlen(argv[3]) > 16) {
        fprintf(stderr, "HOST must be nonempty; NAME must be 1..16 bytes\n");
        return 2;
    }

    /* Synthetic communication ID, scoped separately from real titles.
     * A game adapter replaces this with its actual NP communication ID.
     * p2p_port is 0 because this example has no peer socket. */
    psnr_client* client = psnr_connect(argv[1], port, "RE-PSN-DEMO", argv[3],
                                       0, &user_id, public_ip);
    if (!client) {
        fprintf(stderr, "Connection/HELLO failed\n");
        if (psnr_connect_error())
            fprintf(stderr, "Last HELLO error code (diagnostic hint): %d\n", psnr_connect_error());
        return 1;
    }
    printf("HELLO user=%" PRIu32 "\n", user_id);

    psnr_put16(body, 0);       /* start at the oldest public room */
    psnr_put16(body + 2, 10);  /* at most ten results */
    int status = psnr_call(client, PSNR_SEARCH_ROOMS, body, sizeof(body), &reply, 2000);
    if (status != 1) {
        fprintf(stderr, "SEARCH_ROOMS %s\n", status == 0 ? "timed out" : "disconnected");
        psnr_close(client);
        return 1;
    }
    if (reply.type == PSNR_ERROR) {
        if (reply.len == 4)
            fprintf(stderr, "SEARCH_ROOMS server error %" PRIu32 "\n", psnr_get32(reply.data));
        else
            fprintf(stderr, "Malformed ERROR reply\n");
    } else if (reply.type != PSNR_ROOM_LIST || !room_list_valid(&reply)) {
        fprintf(stderr, "Unexpected or malformed ROOM_LIST reply\n");
    } else {
        printf("ROOM_LIST total=%u count=%u\n", (unsigned)psnr_get16(reply.data),
               (unsigned)psnr_get16(reply.data + 2));
        result = 0;
    }
    psnr_msg_free(&reply);  /* also required for ERROR replies */
    psnr_close(client);
    return result;
}
