<!---
This document describes the transaction graph structure.
-->

# Transaction graph structure

## Transaction entry types

All entries inherit from a common type `AbstractEntry`.

    Header
    Issuance
    Spend
    Output
    Retirement
    Mux
    Data
    Nonce
    Predicate
    TimeRange
    UnknownEntry = all other types, undefined in the current protocol version

### Entry ID

Identifier of the entry is based on its type and body. Body is length-prefixed, type is varint-encoded

    entry.id  = HASH("entryid:" || entry.type  || ":" || entry.body_hash)


## Abstract Entry (base type for all concrete entries)

    - type:     String
    - body:     hashable
    - witness:  hashable

Where `hashable` is any type for which a hashing serialization is defined. See [Serialization for hashing](#serialization-for-hashing) description below.

## UnknownEntry

    - type:         String (not statically known)
    - body_hash:    Hash
    - witness_hash: Hash

**Rules:**

1. Type string must be transmitted explicitly from newer nodes to older nodes, so that the older nodes can compute the Entry ID with the correct type string.

## TxHeader

    entry {
        type="txheader"
        body:
            version:       Integer
            results:       List<Pointer<Output|Retirement|UnknownEntry>>
            data:          Pointer<Data|UnknownEntry>
            mintime:       Integer
            maxtime:       Integer
            ext_hash:      Hash
        witness:
            ext_hash:      Hash        
    }

**Rules:**

1. If version is known, all `ext_hash`es must be hashes of empty strings, `UnknownEntry`s are not allowed in `results`.
2. Results must contain at least one item.
3. Results must of type `Output|Retirement|UnknownEntry`.
4. Every result must be present and valid.
6. Entry `data` must be present and must be of type `Data` or `UnknownEntry`.
7. mintime must be either zero or a timestamp higher than the timestamp of the block that includes the transaction.
8. maxtime must be either zero or a timestamp lower than the timestamp of the block that includes the transaction.


## Data

    entry {
        type="data1"
        body: Hash
        witness: HASH(empty_string)
    }

**Rules:**

1. witness must be empty
2. Note: the body is a hash of the underlying data. The underlying data may not be known. If a transaction author wants to provide the underlying data, it must be done in the transport layer alongside the actual transaction.

### xxx specify hash function for underlying data (w/
domain separation, prob)


## Output

    entry {
        type="output1"
        body:
            source:          ValueSource
            control_program: Program
            data:            Pointer<Data>
            ext_hash:        Hash
        witness:
            ext_hash:        Hash
    }

**Rules:**

1. Let `preventry` be the `source.ref`.
2. `preventry` must be present and valid.
3. Validate `source.amount` is in range.
4. Need to ensure that this output exclusively consumes the `preventry`'s destination.
5. If `preventry` is Spend or Issuance:
    1. Let `dest` be `preventry.destination`.
6. If `preventry` is Mux:
    1. Let `dest` be `preventry.destinations[source.position]`. Fail if not found.
7. Verify previous entry's destination `dest`:
    1. Verify `dest.ref` == `self.id`.
    2. Verify `dest.position` == `0`. This is position of value in this output. TBD: this is double-checked by the input.
8. If tx version is known, all `ext_hash`es must be hashes of empty strings.
9. Insert `self.id` in utxo set.

## Retirement

    entry {
        type="retirement1"
        body:
            source:         ValueSource
            data:           Pointer<Data>
            ext_hash:       Hash
        witness:
            ext_hash:       Hash
    }

**Rules:**

1. Let `preventry` be the `source.ref`.
2. `preventry` must be present and valid.
3. Validate `source.amount` is in range.
4. Need to ensure that this output exclusively consumes the `preventry`'s destination.
5. If `preventry` is Spend or Issuance:
    1. Let `dest` be `preventry.destination`.
6. If `preventry` is Mux:
    1. Let `dest` be `preventry.destinations[source.position]`. Fail if not found.
7. Verify previous entry's destination `dest`:
    1. Verify `dest.ref` == `self.id`.
    2. Verify `dest.position` == `0`. This is position of value in this output. TBD: this is double-checked by the input.
8. If tx version is known, all `ext_hash`es must be hashes of empty strings.



## Spend

    entry {
        type="spend1"
        body:
          spent_output: Pointer<Output>
          data:         Pointer<Data>
          ext_hash:     Hash
        witness:
          destination:  ValueDestination
          arguments:    String
          ext_hash:     Hash        
    }

**Rules:**

1. Validate that `spent_output` is present in tx.
2. Validate that `spent_output` is present in UTXO set.
3. Validate that `destination.ref` has been validated (visited). TBD: !!! figure out what that actually means.
4. If `destination.ref` is a Output or Retirement `nextentry`:
    1. Validate that `destination.position` == `0` (this is value's position in the output). TBD: maybe move this to Output.
    2. Let `src` be `nextentry.source`.
5. If `destination.ref` is a Mux `nextentry`:
    1. Let `src` be `nextentry.sources[destination.position]`. Fail if not present.
6. Validate next entry’s source `src`:
    1. Validate that `src.ref` == `self.id`.
    2. Validate that `src.position` == `0` (this is value's position in the spend).
    3. Validate that `src.value` == `self.spent_output.source.value`.
7. The `spent_output.program` must evaluate to `true` with given `arguments`.
8. Remove `spent_output` from UTXO set.

NB: `spent_output` is not validated, as it was already validated in the transaction that added it to the UTXO.


## Issuance

    entry {
        type="issuance1"
        body:
          anchor:           Pointer<Nonce|Spend>
          value:            AssetAmount
          data:             Pointer<Data>
          ext_hash:         Hash
        witness:
          destination:      ValueDestination
          initial_block_id: Hash
          asset_definition: Pointer<Data>
          issuance_program: Program
          arguments:        String
          ext_hash:         Hash
    }

**Rules:**

1. Check that `value.asset_id == HASH(initial_block_id, asset_definition, issuance_program)`.
2. The `issuance_program` must evaluate to `true` with given `arguments`.
3. Validate that `destination.ref` has been validated (visited). TBD: !!! figure out what that actually means.
4. If `destination.ref` is a Output or Retirement `nextentry`:
    1. Validate that `destination.position` == `0` (this is value's position in the output). TBD: maybe move this to Output.
    2. Let `src` be `nextentry.source`.
5. If `destination.ref` is a Mux `nextentry`:
    1. Let `src` be `nextentry.sources[destination.position]`. Fail if not present.
6. Validate next entry’s source `src`:
    1. Validate that `src.ref` == `self.id`.
    2. Validate that `src.position` == `0` (this is value's position in the input).
    3. Validate that `src.value` == `self.value`.



## Mux

    entry {
        type="mux1"
        body:        
            sources:      List<ValueSource>
            ext_hash:     Hash
        witness:          
            destinations: List<ValueDestination>
            ext_hash:     Hash
    }


**Rules:**

1. Each `source` must be unique in this entry.
2. Each item in `destinations` must be unique in this entry.
3. For each item `dest` in `destinations` at position `i`:  
    1. Validate that `dest.ref` has been validated (visited). TBD !!!!
    2. If `dest.ref` is a Output or Retirement `nextentry`:
        1. Validate that `dest.position` == `0` (this is value's position in the output). TBD: maybe move this to Output.
        2. Let `src` be `nextentry.source`.
    3. If `dest.ref` is a Mux `nextentry`:
        1. Let `src` be `nextentry.sources[dest.position]`. Fail if not present.
    4. Validate next entry’s source `src`:    
        1. Validate that `src.ref` == `self.id`.
        2. Validate that `src.position` == `i` (this is value's position in this mux's destinations list).
        3. Validate that `src.value` == `dest.value`.
        4. Pull the AssetAmount `a` from that `src` and assign it to the `dest`.
4. For each asset ID in the `sources` and `destinations`:
    1. Sum the amounts of that asset ID in `sources` and the amounts of that asset ID in `destinations`.
    2. Test that both the `sources` and `destinations` sums are less than 2^63.
    3. Test that the `sources` sum equals the `destinations` sum.
    4. Check that there is at least one `source` with that asset ID.




## ValueSource

    struct {
        ref:      Pointer<Issuance|Spend|Mux>
        value:    AssetAmount
        position: Integer
    }

**Rules:**

1. Let `entry, src, position` be the current entry, the value source being checked and its position in the entry's sources list.
2. Let `preventry` be the entry referenced by `src.ref`.
3. Let `prevdst` be the value destination located by `preventry.destinations[src.position]`.
4. Verify that `prevdst.ref` equals `entry.id`.
5. Verify that `prevdst.position` equals `position`.
6. Verify that `prevdst.value` equals `src.value`.

## ValueDestination

    struct {
        ref:       Pointer<Output|Retirement|Mux>
        value:     AssetAmount
        position:  Integer
    }

1. Let `entry, dst, position` be the current entry, the value destination being checked and its position in the entry's destinations list.
2. Let `nextentry` be the entry referenced by `dst.ref`.
3. Let `nextsrc` be the value source located by `nextentry.sources[dst.position]`.
4. Verify that `nextsrc.ref` equals `entry.id`.
5. Verify that `nextsrc.position` equals `position`.
6. Verify that `nextsrc.value` equals `dst.value`.


## Nonce

    entry {
        type="nonce1"
        body:
          program:    Program
          timerange:  Pointer<TimeRange>
          ext_hash:   Hash
        witness:
          arguments:  String
          ext_hash:   Hash
    }

**Rules:**

1. If tx version is known, the ext fields must be empty.
2. The ID of the nonce entry must be globally unique on the blockchain.
3. The `program` must evaluate to `true` with given `arguments`.


## TimeRange

    entry {
        type="timerange"
        body:
            mintime:  Integer
            maxtime:  Integer
            ext_hash: Hash
        witness:
            ext_hash: Hash
    }

**Rules:**

1. `mintime` must be equal to or less than the `txheader.mintime` specified in the transaction header.
2. `maxtime` must be equal to or greater than the `txheader.maxtime` specified in the transaction header.


## AssetAmount

This is not a separate entry, but an inlined struct.

    Struct {
        assetID: Hash
        amount:  Integer
    }

## Program

This is not a separate entry, but an inlined struct that carries VM version and the program code in VM-specific encoding.

    Struct {
        vm_version: Integer
        code:       String
    }

## Pointer

Pointer is:

1. encoded as `Hash`,
2. identifies another entry by its ID,
3. restricts the possible acceptable types.

Pointer can be `nil`, in which case it is represented by all-zero 32-byte hash `0x000000...`.

## Serialization for hashing

Primitives:

    Byte
    Hash
    Integer
    String
    List
    Struct

* `Byte` is encoded as 1 byte.
* `Hash` is encoded as 32 bytes.
* `Integer` is encoded as LEB128, with 63-bit limit.
* `String` is encoded as LEB128 length prefix with 31-bit limit followed by raw bytes.
* `List` is encoded as LEB128 length prefix with 31-bit limit followed by serialized items, one by one (`Byte|Hash|Integer|String|List|Struct|ExtStruct`) as defined by the schema.
* `Struct` is encoded as concatenation of all its serialized fields.
* `ExtStruct` is encoded as concatenation of top fields with first `exthash` with a recursive rule for `exthash`. See below.

Common types:

    Pointer = Hash (identifies another entry)
    Program = Struct{vm_version: Integer, code: String}



## ExtStruct

Extensible struct contains flat list of fields and the last field is the extension hash.

The remaining fields are committed to that extension hash. They are defined in flat namespace in a protobuf definition, but for the hashing purposes we group them recursively under another ExtStruct instance:

    ExtStruct {
        fields...
        exthash: Hash
        ext: ExtStruct
    }

    exthash == HASH(ext)
    HASH(ExtStruct) == HASH(serialized-fields || exthash)

Old clients ignore the extended fields (`ext`) and only see the first fields they understand plus the `exthash`.

If the extensibility is not allowed (e.g. when tx version is known), the last `exthash` must be a hash of an empty string.

**Examples:**

V1 schema:

    entry {
        a
        b
        exthash
    }
    ID = H(a || b || exthash1)

V2 schema:

    {
        a
        b
        exthash1
        c
        d
        exthash2
    }
    exthash1 = H(c || d || exthash2)
    ID = H(a || b || exthash1)

Note: when V2 data is encoded and sent to V1, V1 client drops `(c,d,exthash2)` fields and only uses `a,b,exthash1` fields to compute the ID. Which turns out the same as for V2 client.

V3 schema:

    {
        a
        b
        exthash1
        c
        d
        exthash2
        e
        f
        exthash3
    }  
    exthash2 = H(e || f || exthash3)
    exthash1 = H(c || d || exthash2)
    ID = H(a || b || exthash1)

The scheme is applied recursively for the subsequent updates.



## Translation Layer

### OldTx -> NewTx

This is a first intermediate step that allows keeping old SDK, old tx index and data structures within Core, but refactoring how txs and outputs are hashed for UTXO set and merkle root in block headers.

1. Let `oldtx` be the transaction in old format.
2. Let `newtx` be a new instance of `TxHeader` entry.
3. Let `container` be the container for all entries.
4. Set `newtx.version` to `oldtx.version`.
5. If `oldtx.data` is non-empty:
    1. Let `refdata` be a new `Data` entry.
    2. Set `refdata.body` to `tx.data`.
    3. Set `newtx.data` to `refdata.id`.
    4. Add `refdata` to the `container`.
6. Set `newtx.mintime` to `oldtx.mintime`.
7. Set `newtx.maxtime` to `oldtx.maxtime`.
8. Let `mux` be a new `Mux` entry.
9. For each old input `oldinp`:
    1. If the old input is issuance:
        1. Let `is` be a new `Issuance` entry.
        2. Set `is.value` to `AssetAmount { oldinp.assetid, oldinp.amount }`.
        3. If `nonce` is empty:
            1. Set `is.anchor` to the ID of the first new input. (If no input was mapped yet, go back to this step when such input is added.)
        4. If `nonce` is non-empty:
            1. Let `a` be a new `Nonce` entry.
            2. Set `a.program` to (VM1, `PUSHDATA(nonce) DROP ASSET PUSHDATA(oldinp.assetid) EQUAL`. (The program pushes the nonce onto the stack then drops it, then calls the ASSET opcode, pushes the hardcoded old asset ID onto the stack, and checks that they are equal.)
            3. Let `tr` be a new `TimeRange` entry.
            4. Set `tr.mintime` to `oldtx.mintime`.
            5. Set `tr.maxtime` to `oldtx.maxtime`.
            6. Set `a.timerange` to `tr.id`.
            7. Set `is.anchor` to `a.id`.
            8. Add `a` to `container`.
            9. Add `tr` to `container`.
        5. Set `is.initial_block_id` to `oldinp.initial_block_id`.
        6. Set `is.issuance_program` to `oldinp.issuance_program` (with its VM version).
        7. If `oldinp.asset_definition` is non-empty:
            1. Let `adef` be a new `Data` entry.
            2. Set `adef.body` to `oldinp.asset_definition`.
            3. Set `is.asset_definition` to `adef.id`.
            4. Add `adef` to `container`.
        8. If `oldinp.asset_definition` is empty:
            1. Set `is.asset_definition` to a nil pointer `0x000000...`.
        9. Create `ValueSource` struct `src`:
            1. Set `src.ref` to `is.id`.
            2. Set `src.position` to 0.
            3. Set `src.value` to `is.value`.
            4. Add `src` to `mux.sources`.
        10. Add `is` to `container`.
    2. If the old input is a spend:
        1. Let `inp` be a new `Spend` entry.
        2. Set `inp.spent_output` to `oldinp.output_id`.
        3. Set `inp.data` to a nil pointer `0x00000...`.
        4. Create `ValueSource` struct `src`:
            1. Set `src.ref` to `inp.id`.
            2. Set `src.position` to 0.
            3. Set `src.value` to `AssetAmount{ oldinp.spent_output.(assetid,amount) } `.
            4. Add `src` to `mux.sources`.
        5. Add `inp` to `container`.
11. For each output `oldout` at index `i`:
    1. If the `oldout` contains a retirement program:
        1. Let `destentry` be a new `Retirement` entry.
    2. If the `oldout` is not a retirement:
        1. Let `destentry` be a new `Output` entry.
        2. Set `destentry.control_program` to `oldout.control_program` (with its VM version).
    3. Create `ValueSource` struct `src`:
        1. Set `src.ref` to `mux.id`.
        2. Set `src.position` to `i`.
        3. Set `src.value` to `AssetAmount { oldout.asset_id, oldout.amount }`.
        4. Set `destentry.source` to `src`.
    4. If `oldout.data` is non-empty:
        1. Let `data` be a new `Data` entry.
        2. Set `data.body` to `oldout.data`.
        3. Set `destentry.data` to `data.id`.
        4. Add `data` to `container`.
    5. Add `destentry` to `container`.
    6. Add `destentry` to `newtx.results`.


### OldTxID -> NewTxID

1. Map old tx to `newtx`.
2. Return new tx's header ID as NewTxID.

### OldWitTxID -> NewWitTxID

1. Map old tx to new tx.
2. Return new tx's header ID as NewWitTxID. This is the same as NewTxID.

### OldOutputID -> NewOutputID

When indexing old tx's outputs:

1. Map old tx to new tx.
2. Take corresponding new output.
3. Compute its entry ID which will be NewOutputID.
4. Use this new output ID to identify unspent outputs in the DB.

### OldUnspentID -> NewUnspentID

When inserting old tx's outputs into UTXO merkle set:

1. Map old tx to new tx.
2. Take corresponding new output.
3. Compute its entry ID which will be NewUnspentID. (This is the same as NewOutputID.)
4. Use this new unspent ID to insert into UTXO merkle set.


### OldIssuanceHash -> NewIssuanceHash

1. Map old tx to new tx.
2. For each nonce entry in the new tx:
    1. check its time range is within network-defined limits (not unbounded).
    2. Use this entry ID as NewIssuanceHash
    3. Insert new issuance hash in the current _issuance memory_ annotated with expiration date based on `nonce.timerange.maxtime`.

### OldSigHash -> NewSigHash

1. Map old tx to new tx.
2. For each entry where a program is evaluated (Spend, Issuance or Nonce):
    1. Compute `sighash = HASH(txid || entryid)`.



### NewTx -> OldTx

This is a second intermediate step that allows keeping old SDK, but refactoring how txs are represented and stored internally in Core.

TODO: ...


## Compression

1. Serialization prefix indicates the format version that specifies how things are serialized and compressed.
2. Replace hashes with varint offsets, reconstruct hashes in real time and then verify that top hash matches the source (e.g. merkle tree item)
3. Replace some repeated elements such as initial block id with indices too.

### 3. VM mapping

This shows how the implementation of each of the VM instructions need to be changed. Ones that say "no change" will work as already implemented on the OLD data structure.

* CHECKOUTPUT:   no change
* ASSET:         no change
* AMOUNT:        no change
* PROGRAM:       no change
* MINTIME:       no change
* MAXTIME:       no change
* INDEX:         no change
* NONCE:         eliminated
* TXREFDATAHASH: `newtx.refdatahash()`
* REFDATAHASH:   `newcurrentinput.refdatahash()`
* TXSIGHASH:     `hash(newcurrentinput.id() || newtx.id())`
* OUTPUTID:      `newcurrentinput.spent_output.id()`


New opcodes:

* ENTRYID:       `currententry.id()`
* NONCE:         `currentissuance.anchor.id()` (fails if the entry is not an issuance)


### 4. Eliminating witness hash

For simplicity and flexibility, we are removing the commitments to both the transaction witnesses and the block witness.

1. The [transactions Merkle root](https://chain.com/docs/1.1/protocol/specifications/data#transactions-merkle-root) should be calculated based on the transaction IDs, rather than the transaction witness hashes. The code for calculating the transaction witness hashes can be eliminated.

2. `Block ID` (which is the ID included in the next block, and which currently includes the hash of the block witness) should be computed instead to be identical to how the block signature hash is currently computed (i.e., it should use 0x00 serialization flags).


### 5. Block header format

The new slightly different serialization format (i.e. the type prefix and extension hash format) should be applied to the block header as well. We are also removing the block witness from the block ID, as discussed above. Finally, we should flatten the confusing "block commitment" and simply make it three separate fields in the block header.

#### BlockHeader entry

    entry {
        type="blockheader"
        body:
            version:                Integer
            height:                 Integer
            previous_block_id:      Pointer<BlockHeader>
            timestamp:              Integer
            transactions:           MerkleTree<Pointer<TxHeader>>
            assets:                 PatriciaTree<Pointer<Output>>
            next_consensus_program: String
            ext_hash:               Hash
        witness:
            ext_hash:               Hash        
    }

The `MerkleTree` and `PatriciaTree` types are just 32-byte hashes representing the root of those respective trees.

#### OldBlockHeader -> NewBlockHeader

This generates a new BlockHeader data structure, for hashing purposes, from an old block.

1. Let `oldblock` be the block in old format.
2. Let `newblock` be a new instance of `BlockHeader` entry.
3. Set `newblock.version` to `oldblock.version`.
4. Set `newblock.height` to `oldblock.height`.
5. Set `newblock.previous_block_id` to `oldblock.previous_block_id`.
6. Set `newblock.timestamp` to `oldblock.timestamp`.
7. Set `newblock.transactions` to `oldblock.block_commitment.transactions` (i.e. the root of the Merkle tree). Note that this Merkle tree should have been calculated using the new transaction ID.
7. Set `newblock.assets` to `oldblock.block_commitment.assets` (i.e. the root of the Patricia tree). Note that this Patricia tree should have been calculated using the new Output IDs.
8. Set `newblock.next_consensus_program` to `oldblock.block_commitment.next_consensus_program`

#### VM mapping

PROGRAM:       same
NEXTPROGRAM:   same
BLOCKTIME:     same
BLOCKSIGHASH:  newblock.id()
