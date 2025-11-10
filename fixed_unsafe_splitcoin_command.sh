#!/bin/bash
# Fixed unsafe_splitCoin RPC command
# Issue: Empty string "" for optional gas parameter should be null

curl --location 'https://api.testnet.iota.cafe:443' \
--header 'Content-Type: application/json' \
--data '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "unsafe_splitCoin",
  "params": [
    "0xea8180205d4c42f165083d2a42da34b329ac866f8c0eccb9b458bb6a41009296",
    "0x55ed45ebd47a7c315856871b190e35b1457852862fa42aeb002faf37e1bea90a",
    ["1000000000"],
    "0x55ed45ebd47a7c315856871b190e35b1457852862fa42aeb002faf37e1bea90a",
    "1000000000"
  ]
}'

