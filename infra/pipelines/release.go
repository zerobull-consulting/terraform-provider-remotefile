// A generated module for Infra functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/infra/internal/dagger"
)

type Pipelines struct{}

var dotenvVersionTag = "v1.34.0"
var dotenvImage = "dotenv/dotenvx:" + dotenvVersionTag

func (m *Pipelines) dotenvxBinary() *dagger.File {
	return dag.Container().From(dotenvImage).File("/usr/local/bin/dotenvx")
}

var goreleaserVersionTag = "v2.6.1"

func (m *Pipelines) goreleaserBinary() *dagger.File {
	goreleaserImage := "goreleaser/goreleaser:" + goreleaserVersionTag
	return dag.Container().From(goreleaserImage).File("/usr/bin/goreleaser")
}

func (m *Pipelines) Release(ctx context.Context, source *dagger.Directory, dotenvKey *dagger.Secret) (string, error) {
	sourceWithoutBin := source.WithoutDirectory("bin").WithoutDirectory("dist")

	return dag.Container().
		From("golang:1.23-alpine").

		// install git
		WithExec([]string{"apk", "add", "git", "gpg", "gpg-agent", "gnupg"}).

		// use dotenvx to read encrypted sensitive variables like GPG keys
		WithFile("/usr/local/bin/dotenvx", m.dotenvxBinary()).

		// install goreleaser
		WithFile("/usr/local/bin/goreleaser", m.goreleaserBinary()).

		// set the dotenv private key (needed to decrypt .env file)
		WithSecretVariable("DOTENV_PRIVATE_KEY", dotenvKey).

		// copy source code
		WithDirectory("/source", sourceWithoutBin).
		WithWorkdir("/source").

		// Set up GPG for non-interactive use
		WithExec([]string{"mkdir", "-p", "/root/.gnupg"}).
		WithExec([]string{"chmod", "700", "/root/.gnupg"}).
		WithExec([]string{"sh", "-c", "echo 'pinentry-mode loopback' >> /root/.gnupg/gpg.conf"}).
		WithExec([]string{"sh", "-c", "echo 'allow-loopback-pinentry' >> /root/.gnupg/gpg-agent.conf"}).
		WithExec([]string{"sh", "-c", "echo 'no-tty' >> /root/.gnupg/gpg.conf"}).
		
		// import the key
		WithExec([]string{"sh", "-c", "dotenvx get GPG_SECRET_KEY | gpg2 --import --batch"}).
		// and remove the lock file in case gpg2 or gpg-agent didn't clean up properly
		// you may receive "database_open" errors otherwise
		WithExec([]string{"rm", "-f", "/root/.gnupg/public-keys.d/pubring.db.lock"}).
		WithExec([]string{"gpgconf", "--kill", "gpg-agent"}).
		WithExec([]string{"gpg-agent", "--daemon", "--allow-loopback-pinentry"}).

		// run goreleaser
		WithExec([]string{"dotenvx", "run", "-f", ".env", "--", "goreleaser", "release"}).
		Stdout(ctx)
}
