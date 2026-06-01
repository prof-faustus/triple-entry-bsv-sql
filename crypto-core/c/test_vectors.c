// Reproduces the shared crypto-core vectors in C and asserts byte-equality with TS/Go (Appendix B.1).
#include "te_crypto.h"
#include "vectors_gen.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static int fails = 0, checks = 0;

static size_t unhex(const char *h, uint8_t *out) {
  size_t n = strlen(h) / 2;
  for (size_t i = 0; i < n; i++) sscanf(h + 2 * i, "%2hhx", &out[i]);
  return n;
}
static void tohex(const uint8_t *b, size_t n, char *out) {
  for (size_t i = 0; i < n; i++) sprintf(out + 2 * i, "%02x", b[i]);
  out[2 * n] = 0;
}
static void check(const char *what, const uint8_t *got, size_t glen, const char *want) {
  char hx[4096]; tohex(got, glen, hx); checks++;
  if (strcmp(hx, want) != 0) {
    fails++;
    printf("  FAIL %-28s\n    got  %s\n    want %s\n", what, hx, want);
  }
}

int main(void) {
  uint8_t wPriv[32], cPriv[32], wPub[33], cPub[33], tmp[33], cs[33], cs2[33], k[32], gv[32], out[8192];
  unhex(WRITER_PRIV, wPriv); unhex(CP_PRIV, cPriv); unhex(WRITER_PUB, wPub); unhex(CP_PUB, cPub);

  // party pubkeys
  te_pub_from_priv(wPriv, tmp); check("writer.pub", tmp, 33, WRITER_PUB);
  te_pub_from_priv(cPriv, tmp); check("counterparty.pub", tmp, 33, CP_PUB);

  for (int i = 0; i < N_CORE; i++) {
    const core_case_t *c = &CORE_CASES[i];
    uint8_t rowId[256], prevTxid[64], value[1024], blinding[64];
    size_t rl = unhex(c->rowIdHex, rowId), pl = unhex(c->prevTxidHex, prevTxid);
    size_t vl = unhex(c->valueHex, value), bl = unhex(c->blindingHex, blinding);
    te_msg_t m = { c->tableId, rowId, rl, c->columnId, (te_op_t)c->op, strtoull(c->seq, NULL, 10), prevTxid, pl };

    buf_t b; buf_init(&b); te_encode_message(&m, &b);
    check("encodedMessage", b.p, b.len, c->encodedMessage); buf_free(&b);

    te_generator_value(&m, gv); check("gv", gv, 32, c->gv);
    te_sub_pub(wPub, gv, tmp); check("subPubWriter", tmp, 33, c->subW);
    te_sub_pub(cPub, gv, tmp); check("subPubCounterparty", tmp, 33, c->subC);

    te_common_secret(wPriv, cPub, gv, cs); check("cs(writer)", cs, 33, c->cs);
    te_common_secret(cPriv, wPub, gv, cs2); check("cs(counterparty symmetry)", cs2, 33, c->cs);

    te_derive_hmac_key(cs, 33, &m, k); check("kHmac", k, 32, c->kHmac);

    // change image: plaintext = bytes(value); commitment = commit(value, blinding)
    uint8_t img[1100]; size_t il;
    if (c->imageKind == IMG_PLAINTEXT) {
      buf_t ib; buf_init(&ib); w_bytes(&ib, value, vl); il = ib.len; memcpy(img, ib.p, il); buf_free(&ib);
    } else {
      te_commit(value, vl, blinding, bl, img); il = 32;
    }
    check("changeImage", img, il, c->changeImage);

    uint8_t tag[32]; te_hmac_sha256(k, 32, img, il, tag); check("tag", tag, 32, c->tag);

    uint8_t cm[32]; te_commit(value, vl, blinding, bl, cm); check("commit", cm, 32, c->commit);

    buf_t r; buf_init(&r);
    te_encode_record(c->tableId, &m, (te_imgkind_t)c->imageKind, img, il, tag, &r);
    check("encodedRecord", r.p, r.len, c->encodedRecord); buf_free(&r);
  }

  // RFC primitive KATs
  for (int i = 0; i < N_SHA; i++) {
    uint8_t in[256], d[32]; size_t n = unhex(SHA_CASES[i].inputHex, in);
    te_sha256(in, n, d); check(SHA_CASES[i].name, d, 32, SHA_CASES[i].digest);
  }
  for (int i = 0; i < N_HMAC; i++) {
    uint8_t key[256], data[256], mac[32];
    size_t kl = unhex(HMAC_CASES[i].key, key), dl = unhex(HMAC_CASES[i].data, data);
    te_hmac_sha256(key, kl, data, dl, mac); check(HMAC_CASES[i].name, mac, 32, HMAC_CASES[i].mac);
  }
  for (int i = 0; i < N_HKDF; i++) {
    uint8_t ikm[256], salt[256], info[256];
    size_t il = unhex(HKDF_CASES[i].ikm, ikm), sl = unhex(HKDF_CASES[i].salt, salt), nl = unhex(HKDF_CASES[i].info, info);
    te_hkdf_sha256(salt, sl, ikm, il, info, nl, out, HKDF_CASES[i].length);
    check(HKDF_CASES[i].name, out, HKDF_CASES[i].length, HKDF_CASES[i].okm);
  }
  // AEAD
  for (int i = 0; i < N_AEAD; i++) {
    uint8_t key[32], nonce[12], aad[256], pt[256], ct[256], tag[16];
    unhex(AEAD_CASES[i].key, key); unhex(AEAD_CASES[i].nonce, nonce);
    size_t al = unhex(AEAD_CASES[i].aad, aad), pl = unhex(AEAD_CASES[i].plaintext, pt);
    te_aes256gcm_encrypt(key, nonce, aad, al, pt, pl, ct, tag);
    char nm[64]; snprintf(nm, sizeof nm, "%s.ct", AEAD_CASES[i].name); check(nm, ct, pl, AEAD_CASES[i].ciphertext);
    snprintf(nm, sizeof nm, "%s.tag", AEAD_CASES[i].name); check(nm, tag, 16, AEAD_CASES[i].tag);
  }

  printf("C crypto core: %d checks, %d failures\n", checks, fails);
  if (fails == 0) printf("RESULT: C-CORE PARITY PASS (Appendix B.1: C == TS == Go, byte-for-byte)\n");
  return fails ? 1 : 0;
}
