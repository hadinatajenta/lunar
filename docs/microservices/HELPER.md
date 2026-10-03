# Lunar Local Helper

The helper is a small program that runs on the machine where your git repositories live. It reads
git metadata and returns it to the Lunar web interface running in your own browser.

Nothing leaves your computer. The helper only sends data to the browser on the same machine.

## What it reads

Only git metadata:

| Field | Example |
|---|---|
| Repository name | `everest` |
| Current branch | `development` |
| Number of changed files | `6` |
| Time of the last commit | `8 days ago` |
| Last commit hash, author, subject | `88b4ca3`, `Hadinata Jenta`, `fix: guard nil session` |

It never reads file contents. There is no code, no secret, and no document in its response.

## Where it can be reached

The helper binds to `127.0.0.1` only. This means:

- Your own browser can reach it
- Another machine on the same network cannot, even with the correct token
- It is not exposed to the internet

## Getting the helper

Each release publishes prebuilt binaries for these platforms:

| Platform | Artifact name |
|---|---|
| macOS, Apple Silicon | `lunar-helper-darwin-arm64` |
| macOS, Intel | `lunar-helper-darwin-amd64` |
| Linux, x86_64 | `lunar-helper-linux-amd64` |
| Linux, ARM64 | `lunar-helper-linux-arm64` |
| Windows, x86_64 | `lunar-helper-windows-amd64.exe` |

They are built automatically on every push to `main` and attached to the workflow run as artifacts.

## Running it

1. Download the binary for your platform.
2. Make it executable on macOS or Linux:

```
chmod +x lunar-helper-darwin-arm64
```

3. Start it:

```
./lunar-helper-darwin-arm64
```

On Windows, run `lunar-helper-windows-amd64.exe` from a terminal.

Leave the window open. The helper prints its version and the path to its token file.

## The token

On first run the helper creates a random token:

```
~/.lunar/helper-token
```

The directory is created with mode `0700` and the file with mode `0600`, so only your user account
can read it.

The token exists to make sure that only Lunar can call the helper. Any request without it is
rejected with `401 unauthorized`.

To use the helper from Lunar:

1. Run the helper
2. Open Settings, then the Workspace tab
3. Keep the source on **Local helper**
4. Select **Connect helper**. The browser fetches the token from the helper automatically, so there is
   nothing to copy by hand
5. Set the **Repository root folder**, for example `/Users/you/repositories`
6. Select **Validate folder**, then **Save workspace**

If the automatic connection is not possible, the token can still be pasted manually from
`~/.lunar/helper-token` into the Helper token field, then use **Test connection**.

The token is stored encrypted on your Lunar account and is never displayed again. Your browser keeps
it for the current tab session only.

## Configuration

Both variables are optional.

| Variable | Default | Purpose |
|---|---|---|
| `LUNAR_HELPER_ADDR` | `127.0.0.1:5199` | Listen address. Any non-loopback host is forced back to `127.0.0.1` |
| `LUNAR_HELPER_ORIGINS` | `http://localhost:5174`, `http://127.0.0.1:5174` | Allowed browser origins, comma separated. An unlisted origin is rejected with `403` |
| `LUNAR_WORKSPACE_ALLOWED_ROOTS` | empty | Optional allowlist of permitted repository roots. Empty means only the sensitive system paths are rejected |

## Endpoints

Every endpoint except `/health` and `/token` requires the header `Authorization: Bearer <token>`.

`/token` is deliberately reachable without credentials, because the browser does not hold the token
yet at that point. It is protected by two other boundaries: the helper only listens on `127.0.0.1`,
and only origins in the allowlist receive a CORS response. A request from any other website is
rejected with `403`, and no other machine can reach the port at all.

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Liveness and version |
| GET | `/token` | Returns the helper token so the browser can connect in one click |
| GET | `/repositories?root=<path>` | Repository list with lightweight metadata |
| GET | `/services?root=<path>&names=a,b` | Full detail for the named repositories |
| POST | `/validate-path` | Check that a folder exists and is readable, body `{"root_path":"<path>"}` |

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Settings shows the helper as Offline | The helper is not running | Start it, then select **Check again** |
| `401 unauthorized` | Token missing or wrong | Select **Connect helper** again, or paste the current token from `~/.lunar/helper-token` |
| `403` in the browser console | The page origin is not in the allowlist | Add your Lunar origin to `LUNAR_HELPER_ORIGINS` |
| `workspace root does not exist` | The folder is not on this machine, or the path is wrong | Point the root at a folder that exists on this machine |
| Repository listed as `not a git repository` | The folder is not a git repository | This is expected for plain folders. Their checkbox is disabled |

## Why it exists

The Lunar backend usually runs on a shared server. Repositories are private and must stay on the
developer laptop, so the backend cannot read them. The helper closes that gap: the browser talks to
the helper on the same machine, and the shared server never receives repository data.
