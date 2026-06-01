// Emits vectors_gen.h (C test fixtures) from the shared crypto-core vectors.
// Run from crypto-core/c:  node gen_vectors_h.mjs
import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const VEC = resolve(process.cwd(), "..", "vectors");
const core = JSON.parse(readFileSync(resolve(VEC, "core_vectors.json"), "utf8"));
const rfc = JSON.parse(readFileSync(resolve(VEC, "rfc_vectors.json"), "utf8"));
const utf8hex = (s) => Buffer.from(s, "utf8").toString("hex");

let h = `// AUTO-GENERATED from crypto-core/vectors by gen_vectors_h.mjs. Do not edit.
#ifndef TE_VECTORS_GEN_H
#define TE_VECTORS_GEN_H

static const char *WRITER_PRIV = "${core.parties.writer.priv}";
static const char *CP_PRIV     = "${core.parties.counterparty.priv}";
static const char *WRITER_PUB  = "${core.parties.writer.pub}";
static const char *CP_PUB      = "${core.parties.counterparty.pub}";

typedef struct {
  const char *name, *tableId, *rowIdHex, *columnId;
  int op; const char *seq; const char *prevTxidHex;
  const char *valueHex, *blindingHex; int imageKind;
  const char *encodedMessage, *gv, *subW, *subC, *cs, *kHmac, *changeImage, *tag, *commit, *encodedRecord;
} core_case_t;

static const core_case_t CORE_CASES[] = {
`;
for (const c of core.cases) {
  const e = c.expect;
  h += `  { "${c.name}", "${c.message.tableId}", "${c.message.rowId}", "${c.message.columnId}", ${c.message.op}, "${c.message.seq}", "${c.message.prevTxid}", "${c.value}", "${c.blinding}", ${c.imageKind}, "${e.encodedMessage}", "${e.gv}", "${e.subPubWriter}", "${e.subPubCounterparty}", "${e.cs}", "${e.kHmac}", "${e.changeImage}", "${e.tag}", "${e.commit}", "${e.encodedRecord}" },\n`;
}
h += `};
static const int N_CORE = ${core.cases.length};

typedef struct { const char *name, *inputHex, *digest; } sha_case_t;
static const sha_case_t SHA_CASES[] = {
`;
for (const v of rfc.sha256) h += `  { "${v.name}", "${utf8hex(v.input_utf8)}", "${v.digest}" },\n`;
h += `};
static const int N_SHA = ${rfc.sha256.length};

typedef struct { const char *name, *key, *data, *mac; } hmac_case_t;
static const hmac_case_t HMAC_CASES[] = {
`;
for (const v of rfc.hmac_sha256) h += `  { "${v.name}", "${v.key}", "${v.data}", "${v.mac}" },\n`;
h += `};
static const int N_HMAC = ${rfc.hmac_sha256.length};

typedef struct { const char *name, *ikm, *salt, *info; int length; const char *okm; } hkdf_case_t;
static const hkdf_case_t HKDF_CASES[] = {
`;
for (const v of rfc.hkdf_sha256) h += `  { "${v.name}", "${v.ikm}", "${v.salt}", "${v.info}", ${v.length}, "${v.okm}" },\n`;
h += `};
static const int N_HKDF = ${rfc.hkdf_sha256.length};

typedef struct { const char *name, *key, *nonce, *aad, *plaintext, *ciphertext, *tag; } aead_case_t;
static const aead_case_t AEAD_CASES[] = {
`;
for (const v of core.aead) h += `  { "${v.name}", "${v.key}", "${v.nonce}", "${v.aad}", "${v.plaintext}", "${v.ciphertext}", "${v.tag}" },\n`;
h += `};
static const int N_AEAD = ${core.aead.length};

#endif
`;
writeFileSync(resolve(process.cwd(), "vectors_gen.h"), h);
console.log("wrote vectors_gen.h:", core.cases.length, "core,", rfc.sha256.length, "sha,", rfc.hmac_sha256.length, "hmac,", rfc.hkdf_sha256.length, "hkdf,", core.aead.length, "aead");
