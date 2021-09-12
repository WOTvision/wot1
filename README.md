# The WoT blockchain

This is a repository containing an implementation of a blockchain which supports an unique feature set: asserting trust of information organised in JSON documents. 

The goal is to have a global public notary service, for people and corporations to issue signed statements which are globally recognized and verifyable, which are identified by a QR code, which the media and news channels can reproduce and which can be verified by the consumers of information (i.e. ordinary people). The vision is to combat fake news by using a thing blockchains are very good at: distributing reliable data, and by providing a low-tech user-centric interface to it, which everyone can use to verify random junk they hear on the media.

Example use case: The president of a major country issues a (text) statement promising certain actions in the next quarter, regarding an important national topic. He publishes this statement on the WoT blockchain, and this statement automatically receives a QR code. This QR code is reproduced by media (both digital media and classic media, including print) so everyone can verify the statement and read its original wording. The QR code can also be shown in a TV clip of the president giving a public announcement of the statement.

(Current) technical properties of the WoT blockchain (WoT = Web of Trust):

* JSON-based
* Allows publishing of arbitrary JSON documents as transactions
* Uses Ed25519 cryptography
* Allows the use of alphanumeric "handles" instead of addresses
* Allows "upvoting" JSON documents published as transactions
* This upvoting can be used to provide support or endorsement of certain documents, like when a company's president upvotes a statement from their CFO.
* Allows "upvoting" user accounts, to endorse them in a hierarchical manner (e.g. a CEO publishes JSON documents vouching that a certain handle belongs to the company's PR department, the PR department publishes documents vouching that certain YouTube channels or social media accounts belong to the company, etc.)

This blockchain is built primarily to support the feature set described in the following section - however, the blockchain itself is usable for other purposes as well.

**Curent status:** This project is not funded and depends on my diminishing free time. Sponsors and investors are wanted.

# The WoT service v1

For a description what this is all about, read the [White Paper](https://docs.google.com/document/d/1SSBQNTSJY--a-7NjfUMnGdNy4yIg29qOwcWNxHq_DoE/edit?usp=sharing).

Some use cases:

* Politicians can publish digitally signed statements, and reference them in public addresses so they are not taken out of context
* Newspapers can reference politicians' statements, by publishing QR code linking to them
* Companies can publish digitally signed documents, such as investor reports, etc.
* Governments or research institutes can publish digitally signed statistics information, results of R&D, etc.

Current state of project: early development phase / not usable.

## Introduction to publishing on the WoT blockchain

The WoT blockchain supports publishing arbitrary JSON documents as its main feature. In addition to that, and supporting it, the blockchain also implements publishing coin transaction (Bitcoin-style). Publishers (both of documents and of coin transactions) are pseudonymous, i.e. in no case are identifying information required to be present to publish on the WoT blockchain. However, one of the biggest benefits of (optionally) associating identifying information is that it can make documents authenticated, as well as stored immutably and permanently on the blockchain. It is a way for all sorts of entities from ordinary persons, organisations, and even government, to publish authenticated document which are guaranteed to be stored in a provable way.

Publishing on the WoT blockchain begins by publishing an introductory document, which, as the name says, introduces the public key of the publisher, and associates optional identifying information with it. Subsequent documents signed by the same keypair are in this way linked to this identifying information.

Each published document contains the public key of the publisher, and an identification string of the document, which is unique for this publisher. If, at a future point in time, the publisher publishes a document with the same identification string, it is considered to be a newer version of the same document. In this way, keys can be rotated by publishing a new introductory document.

One of the several special fields a document may contain is a statement that the transaction publisher vouches for the validity of a certain transaction (which may contain a document of its own, etc.), which is used to establish a "Web of trust" relationship where publishers choose to verify and vouch for, other publishers and their documents.

## Supported operations

At a high level, the operations supported by the WoT blockchain (in addition to standard cryptocurrency transactions) are:

* Publish a public key and identifying information for it
* Publish a signed JSON document
* Publish a statement that another document (which may e.g. contain an introductory document for a new publisher) is vouched for (or "trusted")
* Rotate keys so that a new key is used for the publisher, maintaining its presence in the blockchain with a new key from this point onwards

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
    "k": "WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY",
    "o": [
        {
            "k": "WF2bn2KvUMR2CJYpekH8wmDZxLj9GoEyREADSZ2I3gkY",
            "a": 1000000,
            "n": 1
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

These are the fields present in the transaction:

* `v` : version number of the transaction format, currently 1
* `f` : transaction flags, e.g. "coinbase"
* `k` : transaction signer public key
* `o` : List of transaction outputs (n=account nonce)
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
* `_vouchtx`: The publisher of this transaction vouches that another transaction contains data he considers valid - he "upvotes" it. The value for the key is the tx hash.

Most of these keys are optional.

**Note:** In the current implementation, the payload JSON document is limited to a single level key-value dictionary where both keys and values are strings.
