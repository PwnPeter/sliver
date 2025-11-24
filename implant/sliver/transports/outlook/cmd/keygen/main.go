package main

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"flag"
	"fmt"
	"os"

	"github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

func main() {
	count := flag.Int("n", 1, "Number of keys to generate")
	format := flag.String("format", "hex", "Output format: hex, base64, json")
	flag.Parse()

	if *count < 1 {
		fmt.Fprintf(os.Stderr, "Error: count must be at least 1\n")
		os.Exit(1)
	}

	for i := 0; i < *count; i++ {
		key, err := outlook.GenerateEncryptionKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
			os.Exit(1)
		}

		switch *format {
		case "hex":
			fmt.Printf("%s\n", outlook.KeyToHex(key))
		case "json":
			if *count == 1 {
				fmt.Printf("{\"encryption_key\": \"%s\"}\n", outlook.KeyToHex(key))
			} else {
				fmt.Printf("{\"key_%d\": \"%s\"}\n", i+1, outlook.KeyToHex(key))
			}
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown format '%s'\n", *format)
			os.Exit(1)
		}
	}
}
