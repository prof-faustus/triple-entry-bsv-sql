#include "te_crypto.h"
#include <stdlib.h>
#include <string.h>
#include <openssl/sha.h>
#include <openssl/hmac.h>
#include <openssl/evp.h>
#include <openssl/kdf.h>
#include <secp256k1.h>

// ---- buffer / Writer (big-endian, length-prefixed) ----
void buf_init(buf_t *b) { b->p = NULL; b->len = 0; b->cap = 0; }
void buf_free(buf_t *b) { free(b->p); buf_init(b); }
static void ensure(buf_t *b, size_t add) {
  if (b->len + add > b->cap) {
    size_t nc = b->cap ? b->cap * 2 : 64;
    while (nc < b->len + add) nc *= 2;
    b->p = realloc(b->p, nc); b->cap = nc;
  }
}
void w_raw(buf_t *b, const uint8_t *p, size_t n) { ensure(b, n); memcpy(b->p + b->len, p, n); b->len += n; }
void w_u8(buf_t *b, uint8_t x) { ensure(b, 1); b->p[b->len++] = x; }
void w_u32(buf_t *b, uint32_t x) { uint8_t t[4] = {x >> 24, x >> 16, x >> 8, x}; w_raw(b, t, 4); }
void w_u64(buf_t *b, uint64_t x) {
  uint8_t t[8]; for (int i = 7; i >= 0; i--) { t[i] = x & 0xff; x >>= 8; } w_raw(b, t, 8);
}
void w_bytes(buf_t *b, const uint8_t *p, size_t n) { w_u32(b, (uint32_t)n); w_raw(b, p, n); }
void w_str(buf_t *b, const char *s) { w_bytes(b, (const uint8_t *)s, strlen(s)); }

// ---- OpenSSL primitives ----
void te_sha256(const uint8_t *d, size_t n, uint8_t out[32]) { SHA256(d, n, out); }
void te_hmac_sha256(const uint8_t *key, size_t kl, const uint8_t *d, size_t n, uint8_t out[32]) {
  unsigned int ol = 32; HMAC(EVP_sha256(), key, (int)kl, d, n, out, &ol);
}
int te_hkdf_sha256(const uint8_t *salt, size_t sl, const uint8_t *ikm, size_t il,
                   const uint8_t *info, size_t inl, uint8_t *out, size_t outlen) {
  EVP_PKEY_CTX *c = EVP_PKEY_CTX_new_id(EVP_PKEY_HKDF, NULL);
  if (!c) return -1;
  int rc = -1;
  if (EVP_PKEY_derive_init(c) <= 0) goto done;
  if (EVP_PKEY_CTX_set_hkdf_md(c, EVP_sha256()) <= 0) goto done;
  if (EVP_PKEY_CTX_set1_hkdf_salt(c, salt, (int)sl) <= 0) goto done;
  if (EVP_PKEY_CTX_set1_hkdf_key(c, ikm, (int)il) <= 0) goto done;
  if (EVP_PKEY_CTX_add1_hkdf_info(c, info, (int)inl) <= 0) goto done;
  if (EVP_PKEY_derive(c, out, &outlen) <= 0) goto done;
  rc = 0;
done:
  EVP_PKEY_CTX_free(c); return rc;
}
int te_aes256gcm_encrypt(const uint8_t key[32], const uint8_t nonce[12], const uint8_t *aad, size_t aadl,
                         const uint8_t *pt, size_t ptl, uint8_t *ct, uint8_t tag[16]) {
  EVP_CIPHER_CTX *c = EVP_CIPHER_CTX_new(); if (!c) return -1;
  int len, rc = -1;
  if (EVP_EncryptInit_ex(c, EVP_aes_256_gcm(), NULL, NULL, NULL) != 1) goto done;
  if (EVP_CIPHER_CTX_ctrl(c, EVP_CTRL_GCM_SET_IVLEN, 12, NULL) != 1) goto done;
  if (EVP_EncryptInit_ex(c, NULL, NULL, key, nonce) != 1) goto done;
  if (aadl && EVP_EncryptUpdate(c, NULL, &len, aad, (int)aadl) != 1) goto done;
  if (EVP_EncryptUpdate(c, ct, &len, pt, (int)ptl) != 1) goto done;
  if (EVP_EncryptFinal_ex(c, ct + len, &len) != 1) goto done;
  if (EVP_CIPHER_CTX_ctrl(c, EVP_CTRL_GCM_GET_TAG, 16, tag) != 1) goto done;
  rc = 0;
done:
  EVP_CIPHER_CTX_free(c); return rc;
}

// ---- encoding ----
void te_encode_message(const te_msg_t *m, buf_t *out) {
  w_raw(out, (const uint8_t *)"TEMC", 4);
  w_u8(out, 1);
  w_str(out, m->table_id);
  w_bytes(out, m->row_id, m->row_id_len);
  w_str(out, m->column_id);
  w_u8(out, (uint8_t)m->op);
  w_u64(out, m->seq);
  w_bytes(out, m->prev_txid, m->prev_txid_len);
}
void te_encode_record(const char *stream_id, const te_msg_t *m, te_imgkind_t kind,
                      const uint8_t *change_image, size_t cil, const uint8_t tag[32], buf_t *out) {
  buf_t menc; buf_init(&menc); te_encode_message(m, &menc);
  w_raw(out, (const uint8_t *)"TER1", 4);
  w_u8(out, 1);
  w_str(out, stream_id);
  w_bytes(out, menc.p, menc.len);
  w_u8(out, (uint8_t)kind);
  w_bytes(out, change_image, cil);
  w_bytes(out, tag, 32);
  buf_free(&menc);
}

// ---- keystone (libsecp256k1) ----
static secp256k1_context *ctx(void) {
  static secp256k1_context *c = NULL;
  if (!c) c = secp256k1_context_create(SECP256K1_CONTEXT_SIGN | SECP256K1_CONTEXT_VERIFY);
  return c;
}
static int ser(const secp256k1_pubkey *pk, uint8_t out[33]) {
  size_t l = 33;
  return secp256k1_ec_pubkey_serialize(ctx(), out, &l, pk, SECP256K1_EC_COMPRESSED) && l == 33 ? 0 : -1;
}
int te_pub_from_priv(const uint8_t priv[32], uint8_t pub[33]) {
  secp256k1_pubkey pk;
  if (!secp256k1_ec_pubkey_create(ctx(), &pk, priv)) return -1;
  return ser(&pk, pub);
}
int te_generator_value(const te_msg_t *m, uint8_t gv[32]) {
  buf_t b; buf_init(&b); te_encode_message(m, &b);
  te_sha256(b.p, b.len, gv); buf_free(&b); return 0;
}
// P2 = P + gv*G  (tweak_add adds gv*G to the pubkey)
int te_sub_pub(const uint8_t pub[33], const uint8_t gv[32], uint8_t out[33]) {
  secp256k1_pubkey pk;
  if (!secp256k1_ec_pubkey_parse(ctx(), &pk, pub, 33)) return -1;
  if (!secp256k1_ec_pubkey_tweak_add(ctx(), &pk, gv)) return -1;
  return ser(&pk, out);
}
// CS = compressed( v2 * P2_other ), v2 = priv + gv mod n
int te_common_secret(const uint8_t priv[32], const uint8_t other_pub[33], const uint8_t gv[32], uint8_t cs[33]) {
  uint8_t v2[32]; memcpy(v2, priv, 32);
  if (!secp256k1_ec_seckey_tweak_add(ctx(), v2, gv)) return -1; // v2 = priv + gv mod n
  secp256k1_pubkey p2;
  if (!secp256k1_ec_pubkey_parse(ctx(), &p2, other_pub, 33)) return -1;
  if (!secp256k1_ec_pubkey_tweak_add(ctx(), &p2, gv)) return -1;   // P2_other = P_other + gv*G
  if (!secp256k1_ec_pubkey_tweak_mul(ctx(), &p2, v2)) return -1;   // CS_point = v2 * P2_other
  return ser(&p2, cs);
}
int te_derive_hmac_key(const uint8_t *cs, size_t csl, const te_msg_t *m, uint8_t k[32]) {
  buf_t salt; buf_init(&salt);
  w_raw(&salt, (const uint8_t *)m->table_id, strlen(m->table_id));
  w_raw(&salt, m->row_id, m->row_id_len);
  w_raw(&salt, (const uint8_t *)m->column_id, strlen(m->column_id));
  buf_t info; buf_init(&info);
  w_raw(&info, (const uint8_t *)"TE/hmac/v1", 10);
  w_u64(&info, m->seq);
  int rc = te_hkdf_sha256(salt.p, salt.len, cs, csl, info.p, info.len, k, 32);
  buf_free(&salt); buf_free(&info); return rc;
}
void te_commit(const uint8_t *value, size_t vl, const uint8_t *r, size_t rl, uint8_t out[32]) {
  buf_t b; buf_init(&b);
  w_raw(&b, (const uint8_t *)"TE/commit/v1", 12);
  w_bytes(&b, r, rl);
  w_bytes(&b, value, vl);
  te_sha256(b.p, b.len, out); buf_free(&b);
}
