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
4. **Both `main` and `develop` already existing on the remote, each with at least one commit.** The test
   config sets `createBranch = "true"`, but that value is passed through to Rundeck's plugin config as-is -
   this resource only calls Setup/Enable to configure and validate the plugin, not an actual export action,
   and Setup's own validation does a real fetch/checkout of the configured branch. It fails immediately with
   "Remote branch not found" if that branch doesn't exist yet, regardless of `createBranch`. The test moves
   from `main` to `develop` partway through (to exercise Update), so both need to exist upfront - neither
   gets created by the test or the provider.
5. Two env vars:
   - `RUNDECK_SCM_TEST_GIT_URL` - SSH-form git URL, e.g. `git@host:org/repo.git`
   - `RUNDECK_SCM_TEST_SSH_KEY_PATH` - local path to the private key, readable from wherever `go test` runs
     (not from inside the Rundeck container - Terraform's `file()` reads it directly and uploads it into
     Rundeck's key storage via `rundeck_private_key`)

## GitHub over SSH requires Rundeck 6.2.1+

Rundeck Enterprise 6.2.0-SNAPSHOT (API v59) failed every `rundeck_scm_export` apply against a real GitHub
remote inside Rundeck's own git-fetch step:

```
400 Bad Request - Failed fetch from the repository:
net.schmizz.sshj.connection.ConnectionException: Stream closed
```

Its server log showed the bundled SSH client (SSHJ 0.40.0) disconnecting itself
(`Disconnected - BY_APPLICATION`) within about a second of the banner exchange, before any authentication was
attempted - consistent with an algorithm-negotiation mismatch between SSHJ 0.40.0 and GitHub's current SSH
server, not a key, network, or credentials problem (confirmed independently: DNS resolved fine, and plain
OpenSSH from inside the same container completed the handshake and authenticated successfully with the same
deploy key, both ed25519 and RSA-4096). This was entirely inside Rundeck's server, not something the
Terraform provider produced or could influence.

**Fixed in Rundeck 6.2.1.** Re-tested `TestAccRundeckScmExport_basic` against a live Rundeck Enterprise 6.2.1
container with the same real GitHub remote and deploy key - passes cleanly, no SSHJ error. If you're
targeting GitHub over SSH specifically, use Rundeck 6.2.1 or later; earlier 6.2.x builds will hit the error
above. This isn't a general requirement for `rundeck_scm_import`/`rundeck_scm_export` themselves (those only
need API v15+, per their own docs) - it's specific to Rundeck's bundled SSH client talking to GitHub's SSH
server.

## Alternative: a self-hosted git server on the same Docker network

If you'd rather not depend on an external GitHub repo and credentials at all, testing against a self-hosted
git server on the same Docker network as Rundeck is more in keeping with this project's `test/oss` pattern
(everything self-contained in Docker, no external dependencies or credentials to manage). Gitea is a good
fit - lightweight, has an SSH git server built in.

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

If you added a GitHub deploy key for this and don't intend to keep testing against it, remove it
(`gh repo deploy-key list --repo <owner>/<repo>` / `gh repo deploy-key delete <id>`) rather than leaving
unused write-access keys around.
