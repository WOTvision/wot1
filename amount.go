package main

// OneCoin is the order of magnitude into which coins are divided. For example,
// if OneCount is 100, it means the coins are sub-divided into 100 pieces of coins.
// Transactions and the database always record amounts in the lowest possible subdivision,
// e.g. "2.54 coins" is recorded as "254".
const OneCoin = 10000

// CoinDecimals is the number of decimal points to which the coin is sub-divided.
// This constant must agree with the OneCoin constant.
const CoinDecimals = 4
