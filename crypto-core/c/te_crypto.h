// te_crypto — C side of the shared TE-BSV crypto core. Byte-exact contract: spec/ALGORITHMS.md.
// Closes Appendix B.1 (C <-> TS/Go parity). Links libsecp256k1 + OpenSSL libcrypto.
#ifndef TE_CRYPTO_H
#define TE_CRYPTO_H
#include <stddef.h>
#include <stdint.h>

// Growable byte buffer (canonical Writer, ALGORITHMS.md §1).
typedef struct { uint8_t *p; size_t len, cap; } buf_t;
void buf_init(buf_t *b);
void buf_free(buf_t *b);
void w_u8(buf_t *b, uint8_t x);
void w_u32(buf_t *b, uint32_t x);
void w_u64(buf_t *b, uint64_t x);
void w_raw(buf_t *b, const uint8_t *p, size_t n);
void w_bytes(buf_t *b, const uint8_t *p, size_t n); // length-prefixed
void w_str(buf_t *b, const char *s);

typedef enum { OP_INSERT = 1, OP_UPDATE = 2, OP_DELETE = 3 } te_op_t;
typedef enum { IMG_PLAINTEXT = 0, IMG_COMMITMENT = 1 } te_imgkind_t;

typedef struct {
  const char *table_id;
  const uint8_t *row_id; size_t row_id_len;
  const char *column_id;
  te_op_t op;
  uint64_t seq;
  const uint8_t *prev_txid; size_t prev_txid_len; // 0 or 32
} te_msg_t;

// primitives (OpenSSL)
void te_sha256(const uint8_t *d, size_t n, uint8_t out[32]);
void te_hmac_sha256(const uint8_t *key, size_t klen, const uint8_t *d, size_t n, uint8_t out[32]);
int  te_hkdf_sha256(const uint8_t *salt, size_t sl, const uint8_t *ikm, size_t il,
                    const uint8_t *info, size_t inl, uint8_t *out, size_t outlen);
int  te_aes256gcm_encrypt(const uint8_t key[32], const uint8_t nonce[12], const uint8_t *aad, size_t aadl,
                          const uint8_t *pt, size_t ptl, uint8_t *ct, uint8_t tag[16]);

// encoding (§1.1/§1.2)
void te_encode_message(const te_msg_t *m, buf_t *out);
void te_encode_record(const char *stream_id, const te_msg_t *m, te_imgkind_t kind,
                      const uint8_t *change_image, size_t cil, const uint8_t tag[32], buf_t *out);

// keystone (§2–§4) — return 0 on success
int te_pub_from_priv(const uint8_t priv[32], uint8_t pub[33]);
int te_generator_value(const te_msg_t *m, uint8_t gv[32]);                 // gv = SHA256(M)
int te_sub_pub(const uint8_t pub[33], const uint8_t gv[32], uint8_t out[33]); // P2 = P + gv*G
int te_common_secret(const uint8_t priv[32], const uint8_t other_pub[33], const uint8_t gv[32], uint8_t cs[33]);
int te_derive_hmac_key(const uint8_t *cs, size_t csl, const te_msg_t *m, uint8_t k[32]);
void te_commit(const uint8_t *value, size_t vl, const uint8_t *r, size_t rl, uint8_t out[32]);

#endif
