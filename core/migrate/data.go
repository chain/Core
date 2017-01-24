package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Type migration describes a single migration.
type migration struct {
	Name      string
	SQL       string
	Hash      string    // set in init
	AppliedAt time.Time // set in loadStatus
}

func init() {
	for i, m := range migrations {
		h := sha256.Sum256([]byte(m.SQL))
		migrations[i].Hash = hex.EncodeToString(h[:])
	}
}

var migrations = []migration{
	{Name: "2016-10-17.0.core.schema-snapshot.sql", SQL: "--\n-- PostgreSQL database dump\n--\n\n-- Dumped from database version 9.5.2\n-- Dumped by pg_dump version 9.5.2\n\nSET statement_timeout = 0;\nSET lock_timeout = 0;\nSET client_encoding = 'UTF8';\nSET standard_conforming_strings = on;\nSET check_function_bodies = false;\nSET client_min_messages = warning;\nSET row_security = off;\n\n--\n-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -\n--\n\nCREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;\n\n\n--\n--\n\n\n\nSET search_path = public, pg_catalog;\n\n--\n-- Name: access_token_type; Type: TYPE; Schema: public; Owner: -\n--\n\nCREATE TYPE access_token_type AS ENUM (\n    'client',\n    'network'\n);\n\n\n--\n-- Name: b32enc_crockford(bytea); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION b32enc_crockford(src bytea) RETURNS text\n    LANGUAGE plpgsql IMMUTABLE\n    AS $$\n\t-- Adapted from the Go package encoding/base32.\n\t-- See https://golang.org/src/encoding/base32/base32.go.\n\t-- NOTE(kr): this function does not pad its output\nDECLARE\n\t-- alphabet is the base32 alphabet defined\n\t-- by Douglas Crockford. It preserves lexical\n\t-- order and avoids visually-similar symbols.\n\t-- See http://www.crockford.com/wrmg/base32.html.\n\talphabet text := '0123456789ABCDEFGHJKMNPQRSTVWXYZ';\n\tdst text := '';\n\tn integer;\n\tb0 integer;\n\tb1 integer;\n\tb2 integer;\n\tb3 integer;\n\tb4 integer;\n\tb5 integer;\n\tb6 integer;\n\tb7 integer;\nBEGIN\n\tFOR r IN 0..(length(src)-1) BY 5\n\tLOOP\n\t\tb0:=0; b1:=0; b2:=0; b3:=0; b4:=0; b5:=0; b6:=0; b7:=0;\n\n\t\t-- Unpack 8x 5-bit source blocks into an 8 byte\n\t\t-- destination quantum\n\t\tn := length(src) - r;\n\t\tIF n >= 5 THEN\n\t\t\tb7 := get_byte(src, r+4) & 31;\n\t\t\tb6 := get_byte(src, r+4) >> 5;\n\t\tEND IF;\n\t\tIF n >= 4 THEN\n\t\t\tb6 := b6 | (get_byte(src, r+3) << 3) & 31;\n\t\t\tb5 := (get_byte(src, r+3) >> 2) & 31;\n\t\t\tb4 := get_byte(src, r+3) >> 7;\n\t\tEND IF;\n\t\tIF n >= 3 THEN\n\t\t\tb4 := b4 | (get_byte(src, r+2) << 1) & 31;\n\t\t\tb3 := (get_byte(src, r+2) >> 4) & 31;\n\t\tEND IF;\n\t\tIF n >= 2 THEN\n\t\t\tb3 := b3 | (get_byte(src, r+1) << 4) & 31;\n\t\t\tb2 := (get_byte(src, r+1) >> 1) & 31;\n\t\t\tb1 := (get_byte(src, r+1) >> 6) & 31;\n\t\tEND IF;\n\t\tb1 := b1 | (get_byte(src, r) << 2) & 31;\n\t\tb0 := get_byte(src, r) >> 3;\n\n\t\t-- Encode 5-bit blocks using the base32 alphabet\n\t\tdst := dst || substr(alphabet, b0+1, 1);\n\t\tdst := dst || substr(alphabet, b1+1, 1);\n\t\tIF n >= 2 THEN\n\t\t\tdst := dst || substr(alphabet, b2+1, 1);\n\t\t\tdst := dst || substr(alphabet, b3+1, 1);\n\t\tEND IF;\n\t\tIF n >= 3 THEN\n\t\t\tdst := dst || substr(alphabet, b4+1, 1);\n\t\tEND IF;\n\t\tIF n >= 4 THEN\n\t\t\tdst := dst || substr(alphabet, b5+1, 1);\n\t\t\tdst := dst || substr(alphabet, b6+1, 1);\n\t\tEND IF;\n\t\tIF n >= 5 THEN\n\t\t\tdst := dst || substr(alphabet, b7+1, 1);\n\t\tEND IF;\n\tEND LOOP;\n\tRETURN dst;\nEND;\n$$;\n\n\n--\n-- Name: cancel_reservation(integer); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION cancel_reservation(inp_reservation_id integer) RETURNS void\n    LANGUAGE plpgsql\n    AS $$\nBEGIN\n    DELETE FROM reservations WHERE reservation_id = inp_reservation_id;\nEND;\n$$;\n\n\n--\n-- Name: create_reservation(text, text, timestamp with time zone, text); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION create_reservation(inp_asset_id text, inp_account_id text, inp_expiry timestamp with time zone, inp_idempotency_key text, OUT reservation_id integer, OUT already_existed boolean, OUT existing_change bigint) RETURNS record\n    LANGUAGE plpgsql\n    AS $$\nDECLARE\n    row RECORD;\nBEGIN\n    INSERT INTO reservations (asset_id, account_id, expiry, idempotency_key)\n        VALUES (inp_asset_id, inp_account_id, inp_expiry, inp_idempotency_key)\n        ON CONFLICT (idempotency_key) DO NOTHING\n        RETURNING reservations.reservation_id, FALSE AS already_existed, CAST(0 AS BIGINT) AS existing_change INTO row;\n    -- Iff the insert was successful, then a row is returned. The IF NOT FOUND check\n    -- will be true iff the insert failed because the row already exists.\n    IF NOT FOUND THEN\n        SELECT r.reservation_id, TRUE AS already_existed, r.change AS existing_change INTO STRICT row\n            FROM reservations r\n            WHERE r.idempotency_key = inp_idempotency_key;\n    END IF;\n    reservation_id := row.reservation_id;\n    already_existed := row.already_existed;\n    existing_change := row.existing_change;\nEND;\n$$;\n\n\n--\n-- Name: expire_reservations(); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION expire_reservations() RETURNS void\n    LANGUAGE plpgsql\n    AS $$\nBEGIN\n    DELETE FROM reservations WHERE expiry < CURRENT_TIMESTAMP;\nEND;\n$$;\n\n\n--\n-- Name: next_chain_id(text); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION next_chain_id(prefix text) RETURNS text\n    LANGUAGE plpgsql\n    AS $$\n\t-- Adapted from the technique published by Instagram.\n\t-- See http://instagram-engineering.tumblr.com/post/10853187575/sharding-ids-at-instagram.\nDECLARE\n\tour_epoch_ms bigint := 1433333333333; -- do not change\n\tseq_id bigint;\n\tnow_ms bigint;     -- from unix epoch, not ours\n\tshard_id int := 4; -- must be different on each shard\n\tn bigint;\nBEGIN\n\tSELECT nextval('chain_id_seq') % 1024 INTO seq_id;\n\tSELECT FLOOR(EXTRACT(EPOCH FROM clock_timestamp()) * 1000) INTO now_ms;\n\tn := (now_ms - our_epoch_ms) << 23;\n\tn := n | (shard_id << 10);\n\tn := n | (seq_id);\n\tRETURN prefix || b32enc_crockford(int8send(n));\nEND;\n$$;\n\n\n--\n-- Name: reserve_utxo(text, bigint, timestamp with time zone, text); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION reserve_utxo(inp_tx_hash text, inp_out_index bigint, inp_expiry timestamp with time zone, inp_idempotency_key text) RETURNS record\n    LANGUAGE plpgsql\n    AS $$\nDECLARE\n    res RECORD;\n    row RECORD;\n    ret RECORD;\nBEGIN\n    SELECT * FROM create_reservation(NULL, NULL, inp_expiry, inp_idempotency_key) INTO STRICT res;\n    IF res.already_existed THEN\n      SELECT res.reservation_id, res.already_existed, res.existing_change, CAST(0 AS BIGINT) AS amount, FALSE AS insufficient INTO ret;\n      RETURN ret;\n    END IF;\n\n    SELECT tx_hash, index, amount INTO row\n        FROM account_utxos u\n        WHERE inp_tx_hash = tx_hash\n              AND inp_out_index = index\n              AND reservation_id IS NULL\n        LIMIT 1\n        FOR UPDATE\n        SKIP LOCKED;\n    IF FOUND THEN\n        UPDATE account_utxos SET reservation_id = res.reservation_id\n            WHERE (tx_hash, index) = (row.tx_hash, row.index);\n    ELSE\n      PERFORM cancel_reservation(res.reservation_id);\n      res.reservation_id := 0;\n    END IF;\n\n    SELECT res.reservation_id, res.already_existed, EXISTS(SELECT tx_hash FROM account_utxos WHERE tx_hash = inp_tx_hash AND index = inp_out_index) INTO ret;\n    RETURN ret;\nEND;\n$$;\n\n\n--\n-- Name: reserve_utxos(text, text, text, bigint, bigint, timestamp with time zone, text); Type: FUNCTION; Schema: public; Owner: -\n--\n\nCREATE FUNCTION reserve_utxos(inp_asset_id text, inp_account_id text, inp_tx_hash text, inp_out_index bigint, inp_amt bigint, inp_expiry timestamp with time zone, inp_idempotency_key text) RETURNS record\n    LANGUAGE plpgsql\n    AS $$\nDECLARE\n    res RECORD;\n    row RECORD;\n    ret RECORD;\n    available BIGINT := 0;\n    unavailable BIGINT := 0;\nBEGIN\n    SELECT * FROM create_reservation(inp_asset_id, inp_account_id, inp_expiry, inp_idempotency_key) INTO STRICT res;\n    IF res.already_existed THEN\n      SELECT res.reservation_id, res.already_existed, res.existing_change, CAST(0 AS BIGINT) AS amount, FALSE AS insufficient INTO ret;\n      RETURN ret;\n    END IF;\n\n    LOOP\n        SELECT tx_hash, index, amount INTO row\n            FROM account_utxos u\n            WHERE asset_id = inp_asset_id\n                  AND inp_account_id = account_id\n                  AND (inp_tx_hash IS NULL OR inp_tx_hash = tx_hash)\n                  AND (inp_out_index IS NULL OR inp_out_index = index)\n                  AND reservation_id IS NULL\n            LIMIT 1\n            FOR UPDATE\n            SKIP LOCKED;\n        IF FOUND THEN\n            UPDATE account_utxos SET reservation_id = res.reservation_id\n                WHERE (tx_hash, index) = (row.tx_hash, row.index);\n            available := available + row.amount;\n            IF available >= inp_amt THEN\n                EXIT;\n            END IF;\n        ELSE\n            EXIT;\n        END IF;\n    END LOOP;\n\n    IF available < inp_amt THEN\n        SELECT SUM(change) AS change INTO STRICT row\n            FROM reservations\n            WHERE asset_id = inp_asset_id AND account_id = inp_account_id;\n        unavailable := row.change;\n        PERFORM cancel_reservation(res.reservation_id);\n        res.reservation_id := 0;\n    ELSE\n        UPDATE reservations SET change = available - inp_amt\n            WHERE reservation_id = res.reservation_id;\n    END IF;\n\n    SELECT res.reservation_id, res.already_existed, CAST(0 AS BIGINT) AS existing_change, available AS amount, (available+unavailable < inp_amt) AS insufficient INTO ret;\n    RETURN ret;\nEND;\n$$;\n\n\nSET default_tablespace = '';\n\nSET default_with_oids = false;\n\n--\n-- Name: access_tokens; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE access_tokens (\n    id text NOT NULL,\n    sort_id text DEFAULT next_chain_id('at'::text),\n    type access_token_type NOT NULL,\n    hashed_secret bytea NOT NULL,\n    created timestamp with time zone DEFAULT now() NOT NULL\n);\n\n\n--\n-- Name: account_control_program_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE account_control_program_seq\n    START WITH 10001\n    INCREMENT BY 10000\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: account_control_programs; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE account_control_programs (\n    id text DEFAULT next_chain_id('acp'::text) NOT NULL,\n    signer_id text NOT NULL,\n    key_index bigint NOT NULL,\n    control_program bytea NOT NULL,\n    change boolean NOT NULL\n);\n\n\n--\n-- Name: account_utxos; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE account_utxos (\n    tx_hash text NOT NULL,\n    index integer NOT NULL,\n    asset_id text NOT NULL,\n    amount bigint NOT NULL,\n    account_id text NOT NULL,\n    control_program_index bigint NOT NULL,\n    reservation_id integer,\n    control_program bytea NOT NULL,\n    metadata bytea NOT NULL,\n    confirmed_in bigint,\n    block_pos integer,\n    block_timestamp bigint,\n    expiry_height bigint\n);\n\n\n--\n-- Name: accounts; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE accounts (\n    account_id text NOT NULL,\n    tags jsonb,\n    alias text\n);\n\n\n--\n-- Name: annotated_accounts; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE annotated_accounts (\n    id text NOT NULL,\n    data jsonb NOT NULL\n);\n\n\n--\n-- Name: annotated_assets; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE annotated_assets (\n    id text NOT NULL,\n    data jsonb NOT NULL,\n    sort_id text NOT NULL\n);\n\n\n--\n-- Name: annotated_outputs; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE annotated_outputs (\n    block_height bigint NOT NULL,\n    tx_pos integer NOT NULL,\n    output_index integer NOT NULL,\n    tx_hash text NOT NULL,\n    data jsonb NOT NULL,\n    timespan int8range NOT NULL\n);\n\n\n--\n-- Name: annotated_txs; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE annotated_txs (\n    block_height bigint NOT NULL,\n    tx_pos integer NOT NULL,\n    tx_hash text NOT NULL,\n    data jsonb NOT NULL\n);\n\n\n--\n-- Name: asset_tags; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE asset_tags (\n    asset_id text NOT NULL,\n    tags jsonb\n);\n\n\n--\n-- Name: assets; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE assets (\n    id text NOT NULL,\n    created_at timestamp with time zone DEFAULT now() NOT NULL,\n    definition_mutable boolean DEFAULT false NOT NULL,\n    sort_id text DEFAULT next_chain_id('asset'::text) NOT NULL,\n    issuance_program bytea NOT NULL,\n    client_token text,\n    initial_block_hash text NOT NULL,\n    signer_id text,\n    definition jsonb,\n    alias text,\n    first_block_height bigint\n);\n\n\n--\n-- Name: assets_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE assets_key_index_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: blocks; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE blocks (\n    block_hash text NOT NULL,\n    height bigint NOT NULL,\n    data bytea NOT NULL,\n    header bytea NOT NULL\n);\n\n\n--\n-- Name: chain_id_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE chain_id_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: config; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE config (\n    singleton boolean DEFAULT true NOT NULL,\n    is_signer boolean,\n    is_generator boolean,\n    blockchain_id text NOT NULL,\n    configured_at timestamp with time zone NOT NULL,\n    generator_url text DEFAULT ''::text NOT NULL,\n    block_xpub text DEFAULT ''::text NOT NULL,\n    remote_block_signers bytea DEFAULT '\\x'::bytea NOT NULL,\n    generator_access_token text DEFAULT ''::text NOT NULL,\n    max_issuance_window_ms bigint,\n    CONSTRAINT config_singleton CHECK (singleton)\n);\n\n\n--\n-- Name: generator_pending_block; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE generator_pending_block (\n    singleton boolean DEFAULT true NOT NULL,\n    data bytea NOT NULL,\n    CONSTRAINT generator_pending_block_singleton CHECK (singleton)\n);\n\n\n--\n-- Name: leader; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE leader (\n    singleton boolean DEFAULT true NOT NULL,\n    leader_key text NOT NULL,\n    expiry timestamp with time zone DEFAULT '1970-01-01 00:00:00-08'::timestamp with time zone NOT NULL,\n    address text NOT NULL,\n    CONSTRAINT leader_singleton CHECK (singleton)\n);\n\n\n--\n-- Name: mockhsm_sort_id_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE mockhsm_sort_id_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: mockhsm; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE mockhsm (\n    pub bytea NOT NULL,\n    prv bytea NOT NULL,\n    alias text,\n    sort_id bigint DEFAULT nextval('mockhsm_sort_id_seq'::regclass) NOT NULL,\n    key_type text DEFAULT 'chain_kd'::text NOT NULL\n);\n\n\n--\n-- Name: pool_tx_sort_id_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE pool_tx_sort_id_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: pool_txs; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE UNLOGGED TABLE pool_txs (\n    tx_hash text NOT NULL,\n    data bytea NOT NULL,\n    sort_id bigint DEFAULT nextval('pool_tx_sort_id_seq'::regclass) NOT NULL\n);\n\n\n--\n-- Name: query_blocks; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE query_blocks (\n    height bigint NOT NULL,\n    \"timestamp\" bigint NOT NULL\n);\n\n\n--\n-- Name: reservation_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE reservation_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: reservations; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE reservations (\n    reservation_id integer DEFAULT nextval('reservation_seq'::regclass) NOT NULL,\n    asset_id text,\n    account_id text,\n    expiry timestamp with time zone DEFAULT '1970-01-01 00:00:00-08'::timestamp with time zone NOT NULL,\n    change bigint DEFAULT 0 NOT NULL,\n    idempotency_key text\n);\n\n\n--\n-- Name: signed_blocks; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE signed_blocks (\n    block_height bigint NOT NULL,\n    block_hash text NOT NULL\n);\n\n\n--\n-- Name: signers; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE signers (\n    id text NOT NULL,\n    type text NOT NULL,\n    key_index bigint NOT NULL,\n    xpubs text[] NOT NULL,\n    quorum integer NOT NULL,\n    client_token text\n);\n\n\n--\n-- Name: signers_key_index_seq; Type: SEQUENCE; Schema: public; Owner: -\n--\n\nCREATE SEQUENCE signers_key_index_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n\n--\n-- Name: signers_key_index_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -\n--\n\nALTER SEQUENCE signers_key_index_seq OWNED BY signers.key_index;\n\n\n--\n-- Name: snapshots; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE snapshots (\n    height bigint NOT NULL,\n    data bytea NOT NULL\n);\n\n\n--\n-- Name: submitted_txs; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE submitted_txs (\n    tx_id text NOT NULL,\n    height bigint NOT NULL,\n    submitted_at timestamp without time zone DEFAULT now() NOT NULL\n);\n\n\n--\n-- Name: txfeeds; Type: TABLE; Schema: public; Owner: -\n--\n\nCREATE TABLE txfeeds (\n    id text DEFAULT next_chain_id('cur'::text) NOT NULL,\n    alias text,\n    filter text,\n    after text,\n    client_token text NOT NULL\n);\n\n\n--\n-- Name: key_index; Type: DEFAULT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY signers ALTER COLUMN key_index SET DEFAULT nextval('signers_key_index_seq'::regclass);\n\n\n--\n-- Name: access_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY access_tokens\n    ADD CONSTRAINT access_tokens_pkey PRIMARY KEY (id);\n\n\n--\n-- Name: account_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY accounts\n    ADD CONSTRAINT account_tags_pkey PRIMARY KEY (account_id);\n\n\n--\n-- Name: account_utxos_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY account_utxos\n    ADD CONSTRAINT account_utxos_pkey PRIMARY KEY (tx_hash, index);\n\n\n--\n-- Name: accounts_alias_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY accounts\n    ADD CONSTRAINT accounts_alias_key UNIQUE (alias);\n\n\n--\n-- Name: annotated_accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY annotated_accounts\n    ADD CONSTRAINT annotated_accounts_pkey PRIMARY KEY (id);\n\n\n--\n-- Name: annotated_assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY annotated_assets\n    ADD CONSTRAINT annotated_assets_pkey PRIMARY KEY (id);\n\n\n--\n-- Name: annotated_outputs_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY annotated_outputs\n    ADD CONSTRAINT annotated_outputs_pkey PRIMARY KEY (block_height, tx_pos, output_index);\n\n\n--\n-- Name: annotated_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY annotated_txs\n    ADD CONSTRAINT annotated_txs_pkey PRIMARY KEY (block_height, tx_pos);\n\n\n--\n-- Name: asset_tags_asset_id_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY asset_tags\n    ADD CONSTRAINT asset_tags_asset_id_key UNIQUE (asset_id);\n\n\n--\n-- Name: assets_alias_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY assets\n    ADD CONSTRAINT assets_alias_key UNIQUE (alias);\n\n\n--\n-- Name: assets_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY assets\n    ADD CONSTRAINT assets_client_token_key UNIQUE (client_token);\n\n\n--\n-- Name: assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY assets\n    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);\n\n\n--\n-- Name: blocks_height_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY blocks\n    ADD CONSTRAINT blocks_height_key UNIQUE (height);\n\n\n--\n-- Name: blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY blocks\n    ADD CONSTRAINT blocks_pkey PRIMARY KEY (block_hash);\n\n\n--\n-- Name: config_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY config\n    ADD CONSTRAINT config_pkey PRIMARY KEY (singleton);\n\n\n--\n-- Name: generator_pending_block_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY generator_pending_block\n    ADD CONSTRAINT generator_pending_block_pkey PRIMARY KEY (singleton);\n\n\n--\n-- Name: leader_singleton_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY leader\n    ADD CONSTRAINT leader_singleton_key UNIQUE (singleton);\n\n\n--\n-- Name: mockhsm_alias_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY mockhsm\n    ADD CONSTRAINT mockhsm_alias_key UNIQUE (alias);\n\n\n--\n-- Name: mockhsm_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY mockhsm\n    ADD CONSTRAINT mockhsm_pkey PRIMARY KEY (pub);\n\n\n--\n-- Name: pool_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY pool_txs\n    ADD CONSTRAINT pool_txs_pkey PRIMARY KEY (tx_hash);\n\n\n--\n-- Name: pool_txs_sort_id_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY pool_txs\n    ADD CONSTRAINT pool_txs_sort_id_key UNIQUE (sort_id);\n\n\n--\n-- Name: query_blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY query_blocks\n    ADD CONSTRAINT query_blocks_pkey PRIMARY KEY (height);\n\n\n--\n-- Name: reservations_idempotency_key_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY reservations\n    ADD CONSTRAINT reservations_idempotency_key_key UNIQUE (idempotency_key);\n\n\n--\n-- Name: reservations_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY reservations\n    ADD CONSTRAINT reservations_pkey PRIMARY KEY (reservation_id);\n\n\n--\n-- Name: signers_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY signers\n    ADD CONSTRAINT signers_client_token_key UNIQUE (client_token);\n\n\n--\n-- Name: signers_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY signers\n    ADD CONSTRAINT signers_pkey PRIMARY KEY (id);\n\n\n--\n-- Name: sort_id_index; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY mockhsm\n    ADD CONSTRAINT sort_id_index UNIQUE (sort_id);\n\n\n--\n-- Name: state_trees_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY snapshots\n    ADD CONSTRAINT state_trees_pkey PRIMARY KEY (height);\n\n\n--\n-- Name: submitted_txs_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY submitted_txs\n    ADD CONSTRAINT submitted_txs_pkey PRIMARY KEY (tx_id);\n\n\n--\n-- Name: txfeeds_alias_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY txfeeds\n    ADD CONSTRAINT txfeeds_alias_key UNIQUE (alias);\n\n\n--\n-- Name: txfeeds_client_token_key; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY txfeeds\n    ADD CONSTRAINT txfeeds_client_token_key UNIQUE (client_token);\n\n\n--\n-- Name: txfeeds_pkey; Type: CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY txfeeds\n    ADD CONSTRAINT txfeeds_pkey PRIMARY KEY (id);\n\n\n--\n-- Name: account_control_programs_control_program_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX account_control_programs_control_program_idx ON account_control_programs USING btree (control_program);\n\n\n--\n-- Name: account_utxos_account_id; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX account_utxos_account_id ON account_utxos USING btree (account_id);\n\n\n--\n-- Name: account_utxos_account_id_asset_id_tx_hash_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX account_utxos_account_id_asset_id_tx_hash_idx ON account_utxos USING btree (account_id, asset_id, tx_hash);\n\n\n--\n-- Name: account_utxos_expiry_height_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX account_utxos_expiry_height_idx ON account_utxos USING btree (expiry_height) WHERE (confirmed_in IS NULL);\n\n\n--\n-- Name: account_utxos_reservation_id_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX account_utxos_reservation_id_idx ON account_utxos USING btree (reservation_id);\n\n\n--\n-- Name: annotated_accounts_jsondata_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_accounts_jsondata_idx ON annotated_accounts USING gin (data jsonb_path_ops);\n\n\n--\n-- Name: annotated_assets_jsondata_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_assets_jsondata_idx ON annotated_assets USING gin (data jsonb_path_ops);\n\n\n--\n-- Name: annotated_assets_sort_id; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_assets_sort_id ON annotated_assets USING btree (sort_id);\n\n\n--\n-- Name: annotated_outputs_jsondata_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_outputs_jsondata_idx ON annotated_outputs USING gin (data jsonb_path_ops);\n\n\n--\n-- Name: annotated_outputs_outpoint_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_outputs_outpoint_idx ON annotated_outputs USING btree (tx_hash, output_index);\n\n\n--\n-- Name: annotated_outputs_timespan_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_outputs_timespan_idx ON annotated_outputs USING gist (timespan);\n\n\n--\n-- Name: annotated_txs_data; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX annotated_txs_data ON annotated_txs USING gin (data);\n\n\n--\n-- Name: assets_sort_id; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX assets_sort_id ON assets USING btree (sort_id);\n\n\n--\n-- Name: query_blocks_timestamp_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX query_blocks_timestamp_idx ON query_blocks USING btree (\"timestamp\");\n\n\n--\n-- Name: reservations_asset_id_account_id_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX reservations_asset_id_account_id_idx ON reservations USING btree (asset_id, account_id);\n\n\n--\n-- Name: reservations_expiry; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX reservations_expiry ON reservations USING btree (expiry);\n\n\n--\n-- Name: signed_blocks_block_height_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE UNIQUE INDEX signed_blocks_block_height_idx ON signed_blocks USING btree (block_height);\n\n\n--\n-- Name: signers_type_id_idx; Type: INDEX; Schema: public; Owner: -\n--\n\nCREATE INDEX signers_type_id_idx ON signers USING btree (type, id);\n\n\n--\n-- Name: account_utxos_reservation_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -\n--\n\nALTER TABLE ONLY account_utxos\n    ADD CONSTRAINT account_utxos_reservation_id_fkey FOREIGN KEY (reservation_id) REFERENCES reservations(reservation_id) ON DELETE SET NULL;\n\n\n--\n-- PostgreSQL database dump complete\n--\n\n"},
	{Name: "2016-10-19.0.core.add-core-id.sql", SQL: "ALTER TABLE config ADD COLUMN id text NOT NULL;\n"},
	{Name: "2016-10-31.0.core.add-block-processors.sql", SQL: `
		CREATE TABLE block_processors (
			name text NOT NULL UNIQUE,
			height bigint DEFAULT 0 NOT NULL
		);
	`},
	{Name: "2016-11-07.0.core.remove-client-token-not-null.sql", SQL: "ALTER TABLE txfeeds ALTER COLUMN client_token DROP NOT NULL;\n"},
	{Name: "2016-11-09.0.utxodb.drop-reservations.sql", SQL: `
		ALTER TABLE account_utxos DROP COLUMN reservation_id;
		DROP TABLE reservations;
		DROP FUNCTION cancel_reservation(integer);
		DROP FUNCTION create_reservation(text, text, timestamp with time zone, text, OUT integer, OUT boolean, OUT bigint);
		DROP FUNCTION expire_reservations();
		DROP FUNCTION reserve_utxo(text, bigint, timestamp with time zone, text);
		DROP FUNCTION reserve_utxos(text, text, text, bigint, bigint, timestamp with time zone, text);
	`},
	{Name: "2016-11-10.0.txdb.drop-pool-txs.sql", SQL: `DROP TABLE pool_txs;`},
	{Name: "2016-11-16.0.account.drop-cp-id.sql", SQL: `
		ALTER TABLE account_control_programs DROP COLUMN id;
		DROP INDEX account_control_programs_control_program_idx;
		ALTER TABLE account_control_programs ADD PRIMARY KEY (control_program);
	`},
	{Name: "2016-11-18.0.account.confirmed-utxos.sql", SQL: `
		DELETE FROM account_utxos WHERE confirmed_in IS NULL;
		ALTER TABLE account_utxos
			DROP COLUMN expiry_height,
			ALTER COLUMN confirmed_in SET NOT NULL,
			ALTER COLUMN block_pos SET NOT NULL,
			ALTER COLUMN block_timestamp SET NOT NULL;
	`},
	{Name: "2016-11-22.0.account.utxos-indexes.sql", SQL: `
		DROP INDEX account_utxos_account_id;
		DROP INDEX account_utxos_account_id_asset_id_tx_hash_idx;
		CREATE INDEX ON account_utxos (asset_id, account_id, confirmed_in);
		ALTER TABLE account_utxos
			DROP COLUMN metadata,
			DROP COLUMN block_pos,
			DROP COLUMN block_timestamp;
	`},
	{Name: "2016-11-23.0.query.jsonb-path-ops.sql", SQL: `
		DROP INDEX annotated_txs_data;
		CREATE INDEX ON annotated_txs USING GIN (data jsonb_path_ops);
	`},
	{Name: "2016-11-28.0.core.submitted-txs-hash.sql", SQL: `
		ALTER TABLE submitted_txs
			ALTER COLUMN tx_id SET DATA TYPE bytea USING decode(tx_id,'hex');
		ALTER TABLE submitted_txs RENAME COLUMN tx_id TO tx_hash;
	`},
	{Name: "2017-01-05.0.core.rename_block_key.sql", SQL: `
		ALTER TABLE config RENAME COLUMN block_xpub TO block_pub;
	`},
	{Name: "2017-01-10.0.signers.xpubs-type.sql", SQL: `
		ALTER TABLE signers ADD COLUMN xpub_byteas bytea[] NOT NULL DEFAULT '{}';
		UPDATE signers s1
			SET xpub_byteas=(SELECT array_agg(decode(unnest(xpubs), 'hex')) FROM signers s2 WHERE s1.id=s2.id);
		ALTER TABLE signers DROP COLUMN xpubs;
		ALTER TABLE signers RENAME COLUMN xpub_byteas TO xpubs;
		ALTER TABLE signers ALTER COLUMN xpubs DROP DEFAULT;
	`},
	{Name: "2017-01-11.0.core.hash-bytea.sql", SQL: `
		ALTER TABLE account_utxos ALTER COLUMN tx_hash SET DATA TYPE bytea USING decode(tx_hash, 'hex');
		ALTER TABLE annotated_outputs ALTER COLUMN tx_hash SET DATA TYPE bytea USING decode(tx_hash, 'hex');
		ALTER TABLE annotated_txs ALTER COLUMN tx_hash SET DATA TYPE bytea USING decode(tx_hash, 'hex');
		ALTER TABLE account_utxos ALTER COLUMN asset_id SET DATA TYPE bytea USING decode(asset_id, 'hex');
		ALTER TABLE asset_tags ALTER COLUMN asset_id SET DATA TYPE bytea USING decode(asset_id, 'hex');
		ALTER TABLE assets ALTER COLUMN id SET DATA TYPE bytea USING decode(id, 'hex');
		ALTER TABLE annotated_assets ALTER COLUMN id SET DATA TYPE bytea USING decode(id, 'hex');
		ALTER TABLE blocks ALTER COLUMN block_hash SET DATA TYPE bytea USING decode(block_hash, 'hex');
		ALTER TABLE signed_blocks ALTER COLUMN block_hash SET DATA TYPE bytea USING decode(block_hash, 'hex');
		ALTER TABLE assets ALTER COLUMN initial_block_hash SET DATA TYPE bytea USING decode(initial_block_hash, 'hex');
		ALTER TABLE config ALTER COLUMN blockchain_id SET DATA TYPE bytea USING decode(blockchain_id, 'hex');
	`},
	{Name: "2017-01-13.0.core.asset-definition-bytea.sql", SQL: `
		ALTER TABLE assets
			ALTER COLUMN definition SET DATA TYPE text;
		ALTER TABLE assets
			ALTER COLUMN definition SET DATA TYPE bytea USING COALESCE(definition::text::bytea, ''),
			ALTER COLUMN definition SET NOT NULL;
		ALTER TABLE assets ADD COLUMN vm_version bigint NOT NULL;
	`},
	{Name: "2017-01-19.0.asset.drop-mutable-flag.sql", SQL: `
		ALTER TABLE assets DROP COLUMN definition_mutable;
	`},
	{Name: "2017-01-20.0.core.add-output-id-to-outputs.sql", SQL: `
		ALTER TABLE annotated_outputs
			ADD COLUMN output_id bytea UNIQUE NOT NULL;
		ALTER TABLE account_utxos
			ADD COLUMN output_id bytea UNIQUE NOT NULL,
			ADD COLUMN unspent_id bytea UNIQUE NOT NULL;
	`},
	{Name: "2017-01-25.0.account.cp-expiry.sql", SQL: `
		ALTER TABLE account_control_programs ADD COLUMN expires_at timestamp with time zone;
	`},
	{Name: "2017-01-30.1.txdb.snapshots-timestamp.sql", SQL: `
		ALTER TABLE snapshots ADD COLUMN created_at timestamp without time zone DEFAULT now();
	`},
	{Name: "2017-01-30.2.core.add-block-hsm-config.sql", SQL: `
		ALTER TABLE config ADD COLUMN block_hsm_url text DEFAULT '',
			ADD COLUMN block_hsm_access_token text DEFAULT '';
	`},
	{Name: "2017-01-30.3.account.remove-unspent-ids.sql", SQL: `
		ALTER TABLE account_utxos DROP COLUMN unspent_id;
	`},
	{Name: "2017-01-31.0.query.drop-outpoint-index.sql", SQL: `
		DROP INDEX annotated_outputs_outpoint_idx;
	`},
	{Name: "2017-01-31.1.query.annotated-schema.sql", SQL: `
		--
		-- Flatten annotated_outputs into schema
		--
		ALTER TABLE annotated_outputs
			ADD COLUMN type text,
			ADD COLUMN purpose text,
			ADD COLUMN asset_id bytea,
			ADD COLUMN asset_alias text,
			ADD COLUMN asset_definition jsonb,
			ADD COLUMN asset_tags jsonb,
			ADD COLUMN asset_local boolean,
			ADD COLUMN amount bigint,
			ADD COLUMN account_id text,
			ADD COLUMN account_alias text,
			ADD COLUMN account_tags jsonb,
			ADD COLUMN control_program bytea,
			ADD COLUMN reference_data jsonb,
			ADD COLUMN local boolean;
		UPDATE annotated_outputs SET
			type             = data->>'type',
			purpose          = COALESCE(data->>'purpose', ''),
			asset_id         = decode(data->>'asset_id', 'hex'),
			asset_alias      = COALESCE(data->>'asset_alias', ''),
			asset_definition = COALESCE(data->'asset_definition', '{}'::jsonb),
			asset_tags       = COALESCE(data->'asset_tags', '{}'::jsonb),
			asset_local      = (data->>'asset_is_local'='yes'),
			amount           = (data->>'amount')::bigint,
			account_id       = data->>'account_id',
			account_alias    = data->>'account_alias',
			account_tags     = data->'account_tags',
			control_program  = decode(data->>'control_program', 'hex'),
			reference_data   = COALESCE(data->'reference_data', '{}'::jsonb),
			local            = (data->>'is_local' = 'yes');
		ALTER TABLE annotated_outputs
			ALTER COLUMN type SET NOT NULL,
			ALTER COLUMN purpose SET NOT NULL,
			ALTER COLUMN asset_id SET NOT NULL,
			ALTER COLUMN asset_alias SET NOT NULL,
			ALTER COLUMN asset_definition SET NOT NULL,
			ALTER COLUMN asset_tags SET NOT NULL,
			ALTER COLUMN asset_local SET NOT NULL,
			ALTER COLUMN amount SET NOT NULL,
			ALTER COLUMN control_program SET NOT NULL,
			ALTER COLUMN reference_data SET NOT NULL,
			ALTER COLUMN local SET NOT NULL,
			DROP COLUMN data;

		--
		-- Flatten annotated_txs into schema
		--
		ALTER TABLE annotated_txs
			ADD COLUMN "timestamp" timestamp with time zone,
			ADD COLUMN block_id bytea,
			ADD COLUMN local boolean,
			ADD COLUMN reference_data jsonb;
		UPDATE annotated_txs SET
			"timestamp"    = (data->>'timestamp')::timestamp with time zone,
			block_id       = decode(data->>'block_id', 'hex'),
			local          = (data->>'is_local' = 'yes'),
			reference_data = COALESCE(data->'reference_data', '{}'::jsonb);
		ALTER TABLE annotated_txs
			ALTER COLUMN timestamp SET NOT NULL,
			ALTER COLUMN block_id SET NOT NULL,
			ALTER COLUMN local SET NOT NULL,
			ALTER COLUMN reference_data SET NOT NULL;

		--
		-- Introduce annotated_inputs
		--
		CREATE TABLE annotated_inputs (
			tx_hash          bytea NOT NULL,
			index            int NOT NULL,
			type             text NOT NULL,
			asset_id         bytea NOT NULL,
			asset_alias      text NOT NULL,
			asset_definition jsonb NOT NULL,
			asset_tags       jsonb NOT NULL,
			asset_local      boolean NOT NULL,
			amount           bigint NOT NULL,
			account_id       text,
			account_alias    text,
			account_tags     jsonb,
			issuance_program bytea NOT NULL,
			reference_data   jsonb NOT NULL,
			local            boolean NOT NULL,
			PRIMARY KEY(tx_hash, index)
		);

		--
		-- Backfill all of the annotated inputs O_O
		--
		INSERT INTO annotated_inputs
		SELECT
			tx_hash,
			idx-1 AS index,
			inp->>'type' AS type,
			decode(inp->>'asset_id', 'hex') AS asset_id,
			COALESCE(inp->>'asset_alias', '') AS asset_alias,
			COALESCE(inp->'asset_definition', '{}'::jsonb) AS asset_definition,
			COALESCE(inp->'asset_tags', '{}'::jsonb) AS asset_tags,
			(inp->>'asset_is_local' = 'yes') AS asset_local,
			(inp->>'amount')::bigint AS amount,
			inp->>'account_id' AS account_id,
			inp->>'account_alias' AS account_alias,
			inp->'account_tags' AS account_tags,
			decode(COALESCE(inp->>'issuance_program', ''), 'hex') AS issuance_program,
			COALESCE(inp->'reference_data', '{}'::jsonb) AS reference_data,
			(inp->>'is_local' = 'yes') AS local
		FROM annotated_txs, jsonb_array_elements(annotated_txs.data->'inputs') WITH ORDINALITY AS inputs (inp, idx);

		--
		-- Flatten annotated_assets into schema.
		--
		ALTER TABLE annotated_assets
			ADD COLUMN alias text,
			ADD COLUMN issuance_program bytea,
			ADD COLUMN keys jsonb,
			ADD COLUMN quorum integer,
			ADD COLUMN definition jsonb,
			ADD COLUMN tags jsonb,
			ADD COLUMN local boolean;
		UPDATE annotated_assets SET
			alias            = COALESCE(data->>'alias', ''),
			issuance_program = decode(COALESCE(data->>'issuance_program', ''), 'hex'),
			keys             = COALESCE(data->'keys', '[]'::jsonb),
			quorum           = (data->>'quorum')::integer,
			definition       = COALESCE(data->'definition', '{}'::jsonb),
			tags             = COALESCE(data->'tags', '{}'::jsonb),
			local            = (data->>'is_local' = 'yes');
		ALTER TABLE annotated_assets
			ALTER COLUMN issuance_program SET NOT NULL,
			ALTER COLUMN keys SET NOT NULL,
			ALTER COLUMN quorum SET NOT NULL,
			ALTER COLUMN definition SET NOT NULL,
			ALTER COLUMN tags SET NOT NULL,
			ALTER COLUMN local SET NOT NULL,
			DROP COLUMN data;

		--
		-- Flatten annotated_accounts into schema.
		--
		ALTER TABLE annotated_accounts
			ADD COLUMN alias text,
			ADD COLUMN keys jsonb,
			ADD COLUMN quorum integer,
			ADD COLUMN tags jsonb;
		UPDATE annotated_accounts SET
			alias  = COALESCE(data->>'alias', ''),
			keys   = COALESCE(data->'keys', '[]'::jsonb),
			quorum = (data->>'quorum')::integer,
			tags   = COALESCE(data->'tags', '{}'::jsonb);
		ALTER TABLE annotated_accounts
			ALTER COLUMN keys SET NOT NULL,
			ALTER COLUMN quorum SET NOT NULL,
			ALTER COLUMN tags SET NOT NULL,
			DROP COLUMN data;
	`},
}
