package main

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
	"os"

	"github.com/wcgcyx/teler/cli"
)

func main() {
	app := cli.NewCLI()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
	}
}
