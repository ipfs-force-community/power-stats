package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
)

var logSystemName = "main"
var log = logging.Logger(logSystemName)

func main() {
	app := &cli.App{
		Name:                 "power-stats",
		Usage:                "stat for filecoin power info about it's implementation",
		Suggest:              true,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "node",
				Usage:    "entry point for a filecoin node, e.g. ws://192.168.200.18:3453/rpc/v1",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "token",
				Usage: "token for a filecoin node",
			},
			&cli.UintFlag{
				Name:  "concurrency",
				Usage: "concurrency for the request to node",
				Value: 100,
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "set log level",
				Value: "error",
			},
		},
		Action:          static,
		HideHelpCommand: true,
		ArgsUsage:       " ",
	}
	app.Setup()
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		return
	}
}

func static(ctx *cli.Context) error {
	// get flag
	nodeUrl := ctx.String("node")
	token := ctx.String("token")
	concurrency := ctx.Uint("concurrency")

	// set log level
	if ctx.IsSet("log-level") {
		if err := logging.SetLogLevel(logSystemName, ctx.String("log-level")); err != nil {
			return fmt.Errorf("set log level error: %w", err)
		}
	}

	// get rpc client
	node, closer, err := newRpcClient(nodeUrl, token)
	if err != nil {
		return fmt.Errorf("build rpc client error: %w", err)
	}
	defer closer()

	// set the throttle
	throttle := make(chan struct{}, concurrency)

	// get miners info
	miners, err := node.StateListMiners(ctx.Context, types.EmptyTSK)
	if err != nil {
		return fmt.Errorf("get miners error: %w", err)
	}

	fmt.Printf("Total %d miners on chain\n", len(miners))
	fmt.Print("It may take a few minutes ...\n\n")

	wg := sync.WaitGroup{}
	wg.Add(len(miners))

	venusQAP, lotusQAP := abi.NewStoragePower(0), abi.NewStoragePower(0)
	for _, miner := range miners {
		throttle <- struct{}{}
		go func(miner address.Address) {
			defer func() {
				<-throttle
				wg.Done()
			}()

			log := log.With("miner", miner)

			// get powerInfo
			powerInfo, err := node.StateMinerPower(ctx.Context, miner, types.EmptyTSK)
			if err != nil {
				log.Errorf("get power: %s", err)
				return
			}
			if !powerInfo.HasMinPower {
				log.Debugf("miner dose not meet min power")
				return
			}
			log = log.With("power", types.SizeStr(powerInfo.MinerPower.QualityAdjPower))

			// get miner info
			minerInfo, err := node.StateMinerInfo(ctx.Context, miner, types.EmptyTSK)
			if err != nil {
				log.Errorf("get miner info: %s", err)
				return
			}
			if minerInfo.PeerId == nil || len(minerInfo.Multiaddrs) == 0 {
				log.Debugf("miner has no peer ID or multiaddrs set on-chain")
				return
			}

			// get agent
			host, err := libp2p.New(libp2p.NoListenAddrs)
			if err != nil {
				log.Errorf("create libp2p host: %s", err)
				return
			}
			defer host.Close()

			var mulAddrs []multiaddr.Multiaddr
			for _, mma := range minerInfo.Multiaddrs {
				ma, err := multiaddr.NewMultiaddrBytes(mma)
				if err != nil {
					log.Warnf("miner had invalid multiaddrs in miner info: %s", err)
				}
				mulAddrs = append(mulAddrs, ma)
			}
			addrInfo := peer.AddrInfo{
				ID:    *minerInfo.PeerId,
				Addrs: mulAddrs,
			}

			if err := host.Connect(ctx.Context, addrInfo); err != nil {
				log.Warnf("connecting to miner: %s", err)
				return
			}

			userAgentI, err := host.Peerstore().Get(addrInfo.ID, "AgentVersion")
			if err != nil {
				log.Warnf("get user agent: %s", err)
				return
			}

			userAgent, ok := userAgentI.(string)
			if !ok {
				log.Errorf("user agent for peer %v was not a string", addrInfo.ID)
				return
			}
			log = log.With("agent", userAgent)

			if isVenus(userAgent) {
				venusQAP = big.Add(venusQAP, powerInfo.MinerPower.QualityAdjPower)
				log.Info("found venus miner")
			} else if isLotus(userAgent) {
				lotusQAP = big.Add(lotusQAP, powerInfo.MinerPower.QualityAdjPower)
				log.Info("found lotus miner")
			}
		}(miner)
	}

	wg.Wait()

	// we need to magnify the proportion to avoid the loss of precision
	var magnification int64 = 100000
	proportionMagnified := big.Div(big.Mul(venusQAP, big.NewInt(magnification)), big.Add(venusQAP, lotusQAP))
	proportionInPercent := 100.0 * float64(proportionMagnified.Int64()) / float64(magnification)

	fmt.Println()
	fmt.Printf("Venus QAP: %s\n", types.SizeStr(venusQAP))
	fmt.Printf("Lotus QAP: %s\n", types.SizeStr(lotusQAP))
	fmt.Printf("Proportion of Venus: %.3f%%\n", proportionInPercent)

	return nil
}

func newRpcClient(endpoint string, token string) (api.FullNode, jsonrpc.ClientCloser, error) {
	requestHeader := http.Header{}
	if token != "" {
		requestHeader.Add("Authorization", "Bearer "+token)
	}
	var res api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), endpoint, "Filecoin", api.GetInternalStructs(&res), requestHeader)
	return &res, closer, err
}

func isLotus(agent string) bool {
	return strings.Contains(agent, "lotus") || strings.Contains(agent, "boost")
}

func isVenus(agent string) bool {
	return strings.Contains(agent, "venus") || strings.Contains(agent, "droplet") || strings.Contains(agent, "market")
}
