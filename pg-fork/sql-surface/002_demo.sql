-- Demo: an ordinary accounting table, journalled with plain SQL (SYS-PG-005), then ordinary
-- INSERT/UPDATE/DELETE — the capture trigger writes the third-entry outbox atomically.

-- A relationship/stream with demo keys (same vector keys as the crypto-core KAT; demo only).
INSERT INTO te.relationship(stream_id, writer_priv, counterparty_pub) VALUES
  ('ledger.acct',
   decode('e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262','hex'),
   decode('0292df7b245b81aa637ab4e867c8d511008f79161a97d64f2ac709600352f7acbc','hex'))
ON CONFLICT (stream_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS public.accounts (
  id      int PRIMARY KEY,
  balance text NOT NULL
);

SELECT te.journal_table('public.accounts', 'ledger.acct', ARRAY['id']);

-- ordinary SQL — no blockchain code visible to the user
INSERT INTO public.accounts(id, balance) VALUES (1, '1000.00');
UPDATE public.accounts SET balance = '1500.00' WHERE id = 1;
INSERT INTO public.accounts(id, balance) VALUES (2, '250.00');
UPDATE public.accounts SET balance = '1499.95' WHERE id = 1;

-- A SECOND stream (SYS-DECIDE-006), CONFIDENTIAL (SYS-HMAC-009): the on-chain third entry carries a
-- blinded commitment, the plaintext stays in the DB. Demonstrates multi-table + multi-stream + confidentiality.
INSERT INTO te.relationship(stream_id, writer_priv, counterparty_pub) VALUES
  ('ledger.hr',
   decode('e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262','hex'),
   decode('0292df7b245b81aa637ab4e867c8d511008f79161a97d64f2ac709600352f7acbc','hex'))
ON CONFLICT (stream_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS public.salaries (
  id     int PRIMARY KEY,
  amount text NOT NULL
);

SELECT te.journal_table('public.salaries', 'ledger.hr', ARRAY['id'], true);  -- confidential

INSERT INTO public.salaries(id, amount) VALUES (1, 'GBP-85000-CONFIDENTIAL');
UPDATE public.salaries SET amount = 'GBP-90000-CONFIDENTIAL' WHERE id = 1;
