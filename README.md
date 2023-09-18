# Filecoin Power Statistics Tool

## Summary

This tool is used to calculate the market share of different filecoin implementations. For more [detail](./docs/venus-market-share-Census.md).

## Usage

### Build

```shell
go build
```

### Run

Run with:
```shell
./power-stats --node=ws://<ip>:<port>/rpc/v1 \
--token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...0UY2Zuf0OGyuLLFXttOwb2EPSyK1745m2qe41EOCN1Q 
```

The output look like this:
```shell
Total 603982 on chain
It may take a few minutes ...


Venus QAP: 1.098 EiB
Lotus QAP: 8.63 EiB
Proportion: 11.287
```

### Options

You can set log level or set concurrency by cli flag. Run `./power-stats --help` for more detail.
