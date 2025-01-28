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

func (m *Pipelines) gpgSecretKey(ctx context.Context, dotenvEncryptedEnvFile *dagger.File, dotenvKey *dagger.Secret) *dagger.Secret {
	// need to run dotenvx and capture the output as a secret

	// for time-savings we use the same image as the dotenv binary
	key, err := dag.Container().From(dotenvImage).
		WithWorkdir("/source").
		WithFile("/source/.env", dotenvEncryptedEnvFile).
		WithSecretVariable("DOTENV_PRIVATE_KEY", dotenvKey).
		WithExec([]string{"dotenvx", "get", "GPG_SECRET_KEY"}).
		Stdout(ctx)

	if err != nil {
		panic(err)
	}

	return dag.SetSecret("GPG_SECRET_KEY", key)
}

// have a remote GPG Agent running and available as a service
func (m *Pipelines) GpgAgentService(ctx context.Context) *dagger.Service {
	return dag.Container().From("alpine:3.21").
		WithExec([]string{"apk", "add", "gnupg", "openssh"}).
		WithExec([]string{"ssh-keygen", "-t", "rsa", "-f", "/etc/ssh/ssh_host_rsa_key"}).
		WithExposedPort(22).AsService(dagger.ContainerAsServiceOpts{Args: []string{"/usr/sbin/sshd", "-D"}})
}

func (m *Pipelines) Release(ctx context.Context, source *dagger.Directory, dotenvKey *dagger.Secret) (int, error) {
	sourceWithoutBin := source.WithoutDirectory("bin").WithoutDirectory("dist")
	gpgKey := m.gpgSecretKey(ctx, source.File(".env"), dotenvKey)

	return dag.Container().
		From("golang:1.23-alpine").

		// install git
		WithExec([]string{"apk", "add", "git", "gpg", "gpg-agent"}).

		// use dotenvx to read encrypted sensitive variables like GPG keys
		WithFile("/usr/local/bin/dotenvx", m.dotenvxBinary()).

		// install goreleaser
		WithFile("/usr/local/bin/goreleaser", m.goreleaserBinary()).

		// set the dotenv private key (needed to decrypt .env file)
		WithSecretVariable("DOTENV_PRIVATE_KEY", dotenvKey).

		// Import the GPG key
		WithMountedSecret("/keys/release-signing-key.asc", gpgKey).
		WithExec([]string{"gpg", "--batch", "--import", "/keys/release-signing-key.asc"}).

		// mount source and run goreleaser
		WithDirectory("/source", sourceWithoutBin).
		WithWorkdir("/source").
		WithExec([]string{"dotenvx", "run", "-f", ".env", "--", "goreleaser", "build"}).
		WithExec([]string{"dotenvx", "run", "-f", ".env", "--", "goreleaser", "archive"}).
		WithExec([]string{"dotenvx", "run", "-f", ".env", "--", "goreleaser", "sign"}).
		WithExec([]string{"dotenvx", "run", "-f", ".env", "--", "goreleaser", "release"}).
		ExitCode(ctx)
}
