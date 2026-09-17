# SCM Integration Testing (`rundeck_scm_import` / `rundeck_scm_export`)

`TestAccRundeckScmExport_basic` (in `rundeck/resource_scm_test.go`) exercises `rundeck_scm_export` against a
real git remote and a real Rundeck server. This isn't Enterprise-gated - git/svn SCM plugins ship with core
Rundeck (API v15+) - but it does need real network infrastructure the other acceptance tests don't, so it's
self-skipping unless two env vars are set.

## What the test needs

1. **A git remote reachable from the Rundeck *server* process itself** - not from wherever `go test` runs.
   If Rundeck is in Docker, the remote has to be reachable from inside that container's network namespace,
   not just from the Docker host.
2. **SSH auth only.** The test config uses `sshPrivateKeyPath`; no HTTPS path is exercised.
3. **A deploy key (or account key) with write/push access** to that remote.
4. **At least one existing commit on the remote's default branch.** The fixture sets `createBranch = "true"`
   for `branch = "main"`, and `createBranch` creates that branch *from* the current default - a genuinely
   empty (zero-commit) repository has no branch to base off of, and setup fails. (This was documented
   incorrectly before - the old instructions said to create an *empty* repo. Fixed.)
5. Two env vars:
   - `RUNDECK_SCM_TEST_GIT_URL` - SSH-form git URL, e.g. `git@host:org/repo.git`
   - `RUNDECK_SCM_TEST_SSH_KEY_PATH` - local path to the private key, readable from wherever `go test` runs
     (not from inside the Rundeck container - Terraform's `file()` reads it directly and uploads it into
     Rundeck's key storage via `rundeck_private_key`)

## Known issue: GitHub doesn't currently work against this Rundeck build

Tried this against a live Rundeck Enterprise 6.2.0-SNAPSHOT (API v59) container with a real private GitHub
repo (`git@github.com:<owner>/<repo>.git`, one commit on `main`, a dedicated deploy key with write access).

Confirmed independently, from inside the Rundeck container:
- DNS resolves `github.com` fine.
- Plain OpenSSH (`ssh -T git@github.com`) completes the handshake and authenticates successfully with the
  deploy key - both an ed25519 key and an RSA-4096 key.

But every `terraform apply` of `rundeck_scm_export` failed inside Rundeck's own git-fetch step:

```
400 Bad Request - Failed fetch from the repository:
net.schmizz.sshj.connection.ConnectionException: Stream closed
```

Rundeck's server log shows its bundled SSH client (SSHJ 0.40.0) disconnecting itself
(`Disconnected - BY_APPLICATION`) within about a second of the banner exchange, before any authentication is
attempted - consistent with an algorithm-negotiation mismatch between SSHJ 0.40.0 and GitHub's current SSH
server, not a key, network, or credentials problem. Identical failure with both key types, ruling out key
type as the cause.

This is generated entirely inside Rundeck's server (the `net.schmizz.sshj` package) - not something the
Terraform provider produces or can influence. **Worth reporting to the Rundeck team** if it reproduces on a
non-snapshot build; in the meantime, don't spend time debugging it further from this repo, and don't assume
a failure against GitHub means the provider is broken - `TestAccProject_localSourceEmptyConfig` (no git
involved) and the rest of the suite pass fine against the same server.

## Recommended path: a self-hosted git server on the same Docker network

Since the SSHJ/GitHub incompatibility is specific to GitHub's SSH server, testing against a self-hosted git
server on the same Docker network as Rundeck sidesteps it entirely, and is more in keeping with this
project's `test/oss` pattern (everything self-contained in Docker, no external dependencies or credentials
to manage). Gitea is a good fit - lightweight, has an SSH git server built in.

Sketch (adjust the network name to match whatever your Rundeck container is actually on -
`docker network ls` / `docker inspect <rundeck-container> --format '{{json .NetworkSettings.Networks}}'`
will show it):

```yaml
# test/enterprise/docker-compose.scm.yml (or add to whatever compose file brings up Rundeck)
services:
  gitea:
    image: gitea/gitea:latest
    environment:
      - GITEA__server__DISABLE_SSH=false
      - GITEA__security__INSTALL_LOCK=true
    ports:
      - "3000:3000"   # web UI, for creating the repo/deploy key from the host
      - "2222:22"     # SSH, for you to push the initial commit from the host
    networks:
      - default        # same network Rundeck is on

networks:
  default:
    name: <rundeck's-network-name>
    external: true
```

Then, from the host (via the mapped `2222` port) or via Gitea's web UI:
1. Create a repo, push one commit to its default branch.
2. Add a deploy key with write access (Gitea supports this natively, same as GitHub).
3. Point the test at Gitea's address **as reachable from inside the Rundeck container** - since Gitea is on
   the same Docker network, that's the Gitea service's container name/port (e.g. `git@gitea:22`), not
   `localhost:2222` (that mapping is only reachable from the host, not from inside another container on the
   same network).

```bash
export RUNDECK_SCM_TEST_GIT_URL="git@gitea:<owner>/<repo>.git"
export RUNDECK_SCM_TEST_SSH_KEY_PATH="/path/to/deploy-key"
go test ./rundeck/... -run TestAccRundeckScmExport_basic -v
```

## Cleanup

The test's own `CheckDestroy` disables the SCM plugin and Terraform destroys the project it creates
(`test-project-scm`) as part of the normal test lifecycle. If a run is interrupted mid-test, clean up
manually:

```bash
curl -X DELETE -H "X-Rundeck-Auth-Token: $RUNDECK_AUTH_TOKEN" \
     "$RUNDECK_URL/api/59/project/test-project-scm"
```

If you added a GitHub deploy key while investigating the issue above and don't intend to keep debugging it,
remove it (`gh repo deploy-key list --repo <owner>/<repo>` / `gh repo deploy-key delete <id>`) rather than
leaving unused write-access keys around.
