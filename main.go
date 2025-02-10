/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/zerobull-consulting/terraform-provider-remotefile/internal/provider"
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "zerobull-consulting/terraform-provider-remotefile",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New("0.2.5"), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
