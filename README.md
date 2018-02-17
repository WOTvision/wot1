# The WoT blockchain app v1

[White Paper](https://docs.google.com/document/d/1SSBQNTSJY--a-7NjfUMnGdNy4yIg29qOwcWNxHq_DoE/edit?usp=sharing)

Current state of project: early development phase / not usable.

## Introduction to publishing on the WoT blockchain

The WoT blockchain supports publishing arbitrary JSON documents as its main feature. In addition to that, and supporting it, the blockchain also implements publishing coin transaction (Bitcoin-style). Publishers (both of documents and of coin transactions) are pseudonymous, i.e. in no case are identifying information required to be present to publish on the WoT blockchain. However, one of the biggest benefits of (optionally) associating identifying information is that it can make documents authenticated, as well as stored immutably and permanently on the blockchain. It is a way for all sorts of entities from ordinary persons, organisations, and even government, to publish authenticated document which are guaranteed to be stored in a provable way.

Publishing on the WoT blockchain begins by publishing an introductory document, which, as the name says, introduces the public key of the publisher, and associates optional identifying information with it. Subsequent documents signed by the same keypair are in this way linked to this identifying information.

Each published document contains the public key of the publisher, and an identification string of the document, which is unique for this publisher. If, at a future point in time, the publisher publishes a document with the same identification string, it is considered to be a newer version of the same document. In this way, keys can be rotated by publishing a new introductory document.

# Technical documentation

## Short intro on how to publish documents from scratch on the WoT blockchain by using the `wot1` app

1. Create a new wallet with the `createwallet` command
2. Obtain enough WoTcoins so that you can publish your documents (TODO: how much?)
3. Publish your intro document transaction with the `publishintro` command
4. Create a JSON document you wish to publish
5. Publish your document with the `publish` command

To publish subsequent documents, only the last 2 steps are needed, if there are enough WoTcoins in the wallet.

## WoT records

The genesis block contains the following transaction:

```json
{
    "v": 1,
    "f": ["coinbase"],
    "i": null,
    "o": [
        {
            "k": "WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY",
            "a": 100000
        }
    ],
    "d": {
        "genesis": "The Guardian, 15th Feb 2018, \"Trump again emphasizes 'mental health' over gun control after Florida shooting\"",
        "comment": "Peace among worlds!",
        "_id": "_intro",
        "_key": "WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY",
        "_name": "WOTvision"
    }
}
```

These are the fields of the JSON document:

* `v` : version number of the transaction format, currently 1
* `f` : transaction flags
* `i` : List of transaction inputs, in the UTXO style
* `o` : List of transaction outputs, in the UTXO style
* `d` : The payload data.

### The payload dictionary

The `d` field of the transaction is the payload document. It is a JSON object which can contain data published to the blockchain, with certain special properties and rules:

* Keys in the first level of the JSON object which begin with an underscore (`_`) are special and reserved. No user-defined key may begin with an underscore in the first level of the JSON object. Examples of special keys in the first level of the payload are the `_id`, `_key` and `_name` keys.
* Keys in the first level of JSON objects are indexed, supporting fast lookup operations.
* Subsequent documents with the same `_key` and `_id` values are considered to update and override earlier documents.

Currently defined special keys in the payload document are:

* `_key`: The public key of the publisher which has published this transaction. This key must verify the transaction signature.
* `_id`: An identifier of the document, unique in the domain of all documents published with the same public key. If a document is published with the same `_key` and `_id` values, it is considered to be a newer version, and a replacement for the same document. Identifiers starting with the underscore (`_`) are reserved, for example the `_intro` identifier.
* `_name`: A human-readable name used in certain types of documents.
* `_newkey`: A new public key the publisher will use from now on. All previously published transactions by this publisher are to be verified with the old key, while all transactions published from now on with this new key are presumed to be associated with the same publisher.
* `_delkey`: Instruction to delete the association between a key and this publisher for all subsequent transactions. I.e. all transactions signed by this particular key will no longer be associated with this publisher.
* `_vouchtx`: The publisher of this transaction vouches that this transaction contains data he considers valid. The value for the key is the tx hash.