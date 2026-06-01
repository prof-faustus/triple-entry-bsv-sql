# C crypto core (Appendix B.1). Requires: build-essential, libssl-dev, libsecp256k1-dev.
CFLAGS = -O2 -Wall -Wextra -std=c11
LIBS = $(shell pkg-config --libs libsecp256k1 libcrypto)

test: vectors_gen.h te_crypto.c test_vectors.c te_crypto.h
	$(CC) $(CFLAGS) te_crypto.c test_vectors.c -o test_vectors $(LIBS)
	./test_vectors

vectors_gen.h:
	node gen_vectors_h.mjs

clean:
	rm -f test_vectors vectors_gen.h

.PHONY: test clean
