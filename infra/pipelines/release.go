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

func (m *Pipelines) dotenvxBinary() *dagger.File {
	dotenvImage := "dotenv/dotenvx:" + dotenvVersionTag
	return dag.Container().From(dotenvImage).File("/usr/local/bin/dotenvx")
}

var goreleaserVersionTag = "v2.6.1"

func (m *Pipelines) goreleaserBinary() *dagger.File {
	goreleaserImage := "goreleaser/goreleaser:" + goreleaserVersionTag
	return dag.Container().From(goreleaserImage).File("/usr/bin/goreleaser")
}

func (m *Pipelines) Release(ctx context.Context, source *dagger.Directory, dotenvKey *dagger.Secret) (int, error) {
	sourceWithoutBin := source.WithoutDirectory("bin").WithoutDirectory("dist")

	return dag.Container().
		From("golang:1.23-alpine").

		// install git
		WithExec([]string{"apk", "add", "git", "gpg"}).

		// use dotenvx to read encrypted sensitive variables like GPG keys
		WithFile("/usr/local/bin/dotenvx", m.dotenvxBinary()).

		// install goreleaser
		WithFile("/usr/local/bin/goreleaser", m.goreleaserBinary()).

		// set the dotenv private key (needed to decrypt .env file)
		WithSecretVariable("DOTENV_PRIVATE_KEY", dotenvKey).

		// mount source and run goreleaser
		WithDirectory("/source", sourceWithoutBin).
		WithWorkdir("/source").
		WithExec([]string{"dotenvx", "run", "-f", ".env", "--", "goreleaser", "release"}).
		ExitCode(ctx)
}
