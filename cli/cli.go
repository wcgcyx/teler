package cli

/*
 * Licensed under LGPL-3.0.
 *
 * You can get a copy of the LGPL-3.0 License at
 *
 * https://www.gnu.org/licenses/lgpl-3.0.en.html
 *
 * @wcgcyx - https://github.com/wcgcyx
 */

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/wcgcyx/teler/version"
)

// NewCLI creates a CLI app.
func NewCLI() *cli.App {
	app := &cli.App{
		Name:      "teler",
		HelpName:  "teler",
		Usage:     "A Trust-based Essential-data Lightweight Ethereum RPC Node",
		UsageText: "teler [global options] command [arguments...]",
		Version:   version.Version,
		Description: "\n\t This is a Trust-based Essential-data Lightweight\n" +
			"\t Ethereum RPC Node or Teler Node.\n\n" +
			"\t It significantly reduces the disk usage by only keeping recent\n" +
			"\t chain state and by relying on trusted source to obtain latest block\n",
		Authors: []*cli.Author{
			{
				Name:  "wcgcyx",
				Email: "wcgcyx@gmail.com",
			},
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:        "start",
			Usage:       "start the teler node syncing process",
			Description: "Start the teler node syncing process",
			ArgsUsage:   " ",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "config",
					Value: "",
					Usage: "specify config file",
				},
				&cli.StringFlag{
					Name:  "blksrc",
					Value: "http://localhost:8545",
					Usage: "specify the trusted block source url",
				},
				&cli.PathFlag{
					Name:  "path",
					Value: "",
					Usage: "specify datastore path",
				},
				&cli.StringFlag{
					Name:  "rpc-host",
					Value: "localhost",
					Usage: "specify eth api rpc service host",
				},
				&cli.IntFlag{
					Name:  "rpc-port",
					Value: 9424,
					Usage: "specify eth api rpc service port",
				},
				&cli.StringFlag{
					Name:  "chain",
					Value: "mainnet",
					Usage: "specify the chain [mainnet,sepolia,holesky]",
				},
			},
			Action: func(ctx *cli.Context) error {
				return runNode(ctx)
			},
		},
		{
			Name:        "version",
			Usage:       "get version",
			Description: "Get the version",
			ArgsUsage:   " ",
			Action: func(c *cli.Context) error {
				fmt.Println("Version: ", version.Version)
				return nil
			},
		},
	}
	return app
}
