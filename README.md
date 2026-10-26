# Lunar

Lunar is one website for your daily developer work. It puts Jira tickets, Bitbucket pull requests,
Confluence documents, your local repositories, and an AI assistant in the same place.

You do not need to be a programmer to start it. If you can copy a command, paste it into a terminal,
and open a browser, you can follow this guide.

Some pages talk to BRI Jira, Bitbucket, and Confluence. You need an account for those systems and a
network connection to them, usually through the company VPN.

---

## Contents

1. [The big picture](#1-the-big-picture)
2. [Before you start](#2-before-you-start)
3. [Start the backend](#3-start-the-backend)
4. [Start the website](#4-start-the-website)
5. [Sign in](#5-sign-in)
6. [Connect your work accounts](#6-connect-your-work-accounts)
7. [Add an AI key](#7-add-an-ai-key)
8. [Connect the helper](#8-connect-the-helper)
9. [A tour of every page](#9-a-tour-of-every-page)
10. [Stop, restart, update, and reset](#10-stop-restart-update-and-reset)
11. [Problems and fixes](#11-problems-and-fixes)
12. [For developers](#12-for-developers)
13. [Safety rules](#13-safety-rules)
14. [Glossary](#14-glossary)
15. [Learn more](#15-learn-more)

---

## Quick start

If you already know how to use a terminal, the whole setup is three terminals.

Terminal 1, the backend:

```bash
cd ~/Code/lunar/backend
cp -n .env.example .env
go mod download
go run ./cmd/server
```

Terminal 2, the website:

```bash
cd ~/Code/lunar/frontend
npm install
npm run dev
```

Terminal 3, the helper:

```bash
cd ~/Code/lunar/backend
go run ./cmd/lunar-helper
```

Open `http://localhost:5174` and sign in with `developer@lunar.dev` and `12345678`.

The rest of this README explains each step slowly, including what to do when something fails.

---

## 1. The big picture

### 1.1 What Lunar does

Lunar is like a control room. Each page is a different window:

| Page | What it helps you do |
|---|---|
| Dashboard | See a summary of your tickets, pull requests, documents, and AI setup |
| Jira | Read your tickets, see your squad's sprint, and create branches for an issue |
| Bitbucket | See your recent pushes, open pull requests, and review code with AI |
| Confluence | Read technical documents and generate query reviews, unit test documents, and SOPs |
| Microservices | See the state of your repositories and export a Postman collection |
| Copilot | Ask questions about your code and documents |
| Settings | Store your tokens, AI keys, and workspace folder |

### 1.2 The three pieces

Lunar is made of three small programs. Think of them as three people in an office:

| Piece | Port | Job | Where it runs |
|---|---|---|---|
| Frontend | `5174` | The face. This is the website you see in the browser | Your computer |
| Backend | `8081` | The brain. It talks to Jira, Bitbucket, Confluence, and AI providers | Your computer or a server |
| Helper | `5199` | The hands. It reads git repositories on your computer | Your computer only |

The helper is optional. Without it, chat still works, but Lunar cannot read repositories on your
computer and the code-aware Copilot cannot answer with code context.

```text
Your browser
    |
    |----> Frontend (5174) ----> Backend (8081) ----> Jira, Bitbucket, Confluence, AI providers
    |                                  |
    |                                  +----> SQLite database: users, encrypted tokens, chat history
    |
    +----> Helper (5199, only on 127.0.0.1) ----> git repositories and the local code index
```

The helper only listens on `127.0.0.1`. Another computer cannot reach it, even on the same Wi-Fi.

### 1.3 What works without the helper

| Feature | Without the helper |
|---|---|
| Sign in, Dashboard, Settings | Works |
| Jira | Works if the backend can reach Jira |
| Bitbucket | Works if the backend can reach Bitbucket |
| Confluence | Works if the backend can reach Confluence |
| Copilot chat | Works through the backend |
| Copilot with code context | Does not work |
| Microservices board, when the repositories are on your computer | Does not work |
| Unit test documents and SOP generation | Does not work |
| Postman export from repositories on your computer | Does not work |

The Microservices board and the Postman export can also read repositories through the backend, using
the **On the server** source. That only works when the repositories are on the machine that runs the
backend. For repositories on your own laptop, use the helper.

---

## 2. Before you start

### 2.1 Checklist

You need:

- A computer with macOS, Linux, or Windows.
- A terminal. On macOS, press `Command + Space`, type `Terminal`, and press `Enter`.
- Go 1.24.1 or newer.
- Node.js 22.18 or newer. Node.js 22 LTS is a good choice.
- npm, which is installed together with Node.js.
- Git, which is installed on most developer computers.
- Access to BRI Jira, Bitbucket, and Confluence, usually through VPN.
- An AI provider key, if you want to use Copilot, AI review, or AI document generation.

You do **not** need Docker, a database server, or a cloud account. Lunar stores its data in a local
SQLite file.

This guide uses macOS and Linux commands. On Windows, use PowerShell. Most commands are the same, but
`cp` becomes `Copy-Item`, `cat` becomes `Get-Content`, and `rm -rf` becomes
`Remove-Item -Recurse -Force`.

### 2.2 Check Go

Run:

```bash
go version
```

You should see something like:

```text
go version go1.24.1 darwin/arm64
```

Any version `1.24.1` or newer is fine. If the command is not found, install Go from
`https://go.dev/dl/`, then open a new terminal and try again.

### 2.3 Check Node.js and npm

Run:

```bash
node --version
npm --version
```

You should see a Node.js version like `v22.18.0` or newer, and an npm version like `10.x` or newer.
If Node.js is missing or too old, install the LTS version from `https://nodejs.org/`.

### 2.4 Check Git

Run:

```bash
git --version
```

You should see a version like `git version 2.39.5`. Install Git if the command is not found.

### 2.5 Get the code

If the `lunar` folder is already on your computer, skip this step.

```bash
cd ~/Code
git clone https://github.com/hadinatajenta/lunar.git
cd lunar
```

This guide uses `~/Code/lunar` as the folder. If your folder is somewhere else, use your real path in
every command.

From now on, this guide says "the Lunar folder" for that folder.

---

## 3. Start the backend

The backend is the program that talks to Jira, Bitbucket, Confluence, and the AI providers. It also
stores users, encrypted tokens, workspace settings, and chat history.

### 3.1 Open a terminal

Open a terminal and go to the backend folder:

```bash
cd ~/Code/lunar/backend
```

The prompt should now show that you are inside the `backend` folder. If you are not sure, run:

```bash
pwd
```

You should see a path ending in `/lunar/backend`.

### 3.2 Create your settings file

The repository ships an example settings file. Copy it without overwriting an existing file:

```bash
cp -n .env.example .env
```

If `backend/.env` already exists, the command leaves it alone. It may contain your team's real keys.

The `.env` file is private. Never commit it, paste it into a chat, or send it to another person.

### 3.3 Understand the settings file

Open `backend/.env` in any text editor. On macOS, run `open -e .env` to edit it in TextEdit. On
Linux, run `nano .env`. The important values are:

| Name | What it means |
|---|---|
| `PORT` | The backend port. Keep `8081` unless it is already used |
| `DB_PATH` | Where the SQLite file lives. The default is `data/lunar.db` |
| `JWT_SECRET` | The secret used to sign login sessions. Change it for a real deployment |
| `ENCRYPTION_KEY` | The key used to encrypt stored tokens. 32 characters is recommended |
| `CORS_ORIGIN` | Which website may call the backend. `*` is fine for local development only |
| `LUNAR_WORKSPACE_ALLOWED_ROOTS` | Folders the backend and helper may modify. Required for git write actions |
| `JIRA_BASE_URL` | The Jira address. The default is the BRI Jira address |
| `BITBUCKET_BASE_URL` | The Bitbucket address |
| `CONFLUENCE_BASE_URL` | The Confluence address |
| `SYSTEM_DEEPSEEK_KEY` | Optional shared DeepSeek key for everyone |
| `SYSTEM_GEMINI_KEY` | Optional shared Gemini key for everyone |
| `SYSTEM_OPENAI_KEY` | Optional shared OpenAI key for everyone |
| `SYSTEM_CLAUDE_KEY` | Optional shared Claude key for everyone |
| `SYSTEM_MIMO_KEY` | Optional shared Xiaomi MiMo key for everyone |
| `SYSTEM_OPENCODE_GO_KEY` | Optional shared OpenCode Go key for everyone |
| `SEED_USER_EMAIL` | The first account created automatically |
| `SEED_USER_PASSWORD` | The password of that first account |
| `SEED_USER_NAME` | The display name of that first account |

The example file already contains working local values. You can keep them for a first run.

Two rules matter:

1. `ENCRYPTION_KEY` must stay the same after you save tokens. If you change it, Lunar cannot read the
   tokens it saved before.
2. `JWT_SECRET` and `ENCRYPTION_KEY` in the example are public. Change them before you share a
   deployment with other people.

You can generate new values with:

```bash
openssl rand -hex 32
openssl rand -hex 16
```

The first command produces a good `JWT_SECRET`. The second produces exactly 32 characters for
`ENCRYPTION_KEY`. A different non-empty value also works, because Lunar hashes it, but a stable
32-character value is easier to reason about.

### 3.4 Download the backend packages

Stay in the backend folder and run:

```bash
go mod download
```

This downloads the Go libraries Lunar needs. It can take a minute the first time.

### 3.5 Start the backend

Run:

```bash
go run ./cmd/server
```

You should see:

```text
server starting on port 8081
```

Leave this terminal open. The backend must keep running while you use Lunar.

The first start also creates:

- `backend/data/lunar.db`, the SQLite database.
- A first user, using `SEED_USER_EMAIL` and `SEED_USER_PASSWORD`.
- All database tables.

With the example values, that first user is:

```text
Email: developer@lunar.dev
Password: 12345678
```

If you change the seed values later, the old user stays in the database. The server adds a new user
only when the new email does not exist yet.

### 3.6 Check that the backend is alive

Open a second terminal, or use the same terminal after the backend is running in another window:

```bash
curl http://localhost:8081/api/health
```

You should see:

```json
{"status":"healthy"}
```

If you see `Connection refused`, the backend is not running, or it is running on a different port.

---

## 4. Start the website

The website is the part you see. It runs separately from the backend.

### 4.1 Open a second terminal

Do not stop the backend. Open a new terminal window or tab.

Go to the frontend folder:

```bash
cd ~/Code/lunar/frontend
```

### 4.2 Install the website packages

Run:

```bash
npm install
```

This downloads Vue, Vite, and the other frontend libraries. It can take a few minutes the first time.

The repository includes `package-lock.json`, so npm is the simplest choice. If your team uses pnpm,
`pnpm install` and `pnpm dev` also work.

### 4.3 Start the website

Run:

```bash
npm run dev
```

You should see:

```text
Local: http://localhost:5174/
```

Open that address in your browser:

```text
http://localhost:5174
```

Leave this terminal open too. If you close it, the website stops.

Do not run `npm run build` for normal use. That command creates a production build and is only needed
before a deployment.

### 4.4 What the frontend does with the backend

The frontend sends every request that starts with `/api` to `http://localhost:8081`. That rule lives
in `frontend/vite.config.ts`.

If you change the backend port, change that proxy target too.

### 4.5 If Vite chooses another port

If port `5174` is already used, Vite may start on `5175` instead. The helper only accepts `5174` by
default, so connecting the helper will fail.

Use one of these fixes:

- Stop the other program that uses `5174`.
- Start Vite on `5174` after the other program stops.
- Add the new address to `LUNAR_HELPER_ORIGINS` in `backend/.env`, then restart the helper.

---

## 5. Sign in

Open `http://localhost:5174`. If you are not signed in, Lunar sends you to the sign-in page.

### 5.1 Use the first account

With the example values, sign in with:

```text
Email: developer@lunar.dev
Password: 12345678
```

### 5.2 Create your own account

1. Click **Create account** or open `http://localhost:5174/register`.
2. Paste your **Jira Personal Access Token**.
3. Click **Verify Identity**.
4. Check that the name, email, and username are yours.
5. Create a password with at least 6 characters.
6. Repeat the password.
7. Click **Create Workspace Account**.

A Jira token is required because Lunar checks your identity against BRI Jira. If the token is wrong,
or you are not on the company network, verification fails.

If your BRI email already has an account, sign in instead. Lunar does not allow two accounts with the
same BRI identity.

### 5.3 Forgot password

The **Forgot password** link only sends you back to the sign-in page. There is no password reset flow
yet. If you lose the seeded password, you can create another account with a Jira token, or ask the
person who runs the backend to reset the database.

### 5.4 What the login token means

After sign-in, the browser keeps a session token. It is sent with every backend request and expires
after 24 hours. When it expires, sign in again.

To sign out, use the sign-out button in the sidebar. Do this when you finish on a shared computer.

---

## 6. Connect your work accounts

Lunar needs permission to read Jira, Bitbucket, and Confluence. You give it personal access tokens,
often called PATs.

Open **Settings**, then the **Integrations** tab.

### 6.1 Jira

1. Paste your **Jira Personal Access Token**.
2. Click **Save changes**.
3. The status should change to **Configured**.

The Jira base URL is read from `backend/.env`. It is shown as a read-only field.

### 6.2 Bitbucket

1. Enter your Bitbucket **username**. It usually looks like `0099999`.
2. Paste your **Bitbucket Personal Access Token**.
3. Click **Save changes**.

Bitbucket needs the username because the token alone does not identify the account in every call.

### 6.3 Confluence

1. Paste your **Confluence Personal Access Token**.
2. Click **Save changes**.

Confluence uses the same Atlassian account as Jira in most cases, but it still needs its own token
entry.

### 6.4 What happens to the tokens

Lunar encrypts the tokens with AES-256-GCM and stores them on the backend. The browser never receives
the plain token again. If you leave a token field blank and save, Lunar keeps the old token.

To replace a token, paste the new one and save. This version has no remove button, so ask the
backend administrator to clear a token you no longer want.

### 6.5 If a token is rejected

A rejected token is almost always one of these:

- The token was copied with an extra space or a missing character.
- The token expired.
- The token does not have the permission Lunar needs.
- The backend cannot reach the company system because VPN is off.

Ask your Jira, Bitbucket, or Confluence administrator which permissions your token needs.

---

## 7. Add an AI key

The AI features need a model. You can use a personal key, or a shared key that the backend
administrator placed in `.env`.

Open **Settings**, then the **AI providers** tab.

### 7.1 Choose a provider

Lunar supports:

| Provider | Notes |
|---|---|
| OpenAI | Pay per request |
| Anthropic Claude | Pay per request |
| Google Gemini | Pay per request |
| DeepSeek | Pay per request |
| Xiaomi MiMo | Pay per request |
| OpenCode Go | A subscription with open models. It does not serve Claude models |

You only need one working provider to start.

### 7.2 Save a key

1. Find the provider row.
2. Paste the key into its field.
3. Click the save action for that provider.
4. The row should show **Configured**.

If you see **System** or a system-provided label, the backend already has a shared key for that
provider and you do not need your own.

### 7.3 Where the key appears

When you open Copilot, the model picker only lists models whose provider has a key. If no provider has
a key, the picker is empty and AI actions are disabled or show a lock.

The key is encrypted at rest and is never shown again. Leave the field blank to keep the saved key.

### 7.4 If an AI request fails

Check, in this order:

1. The provider key is correct and still has credit.
2. The computer can reach the provider. Some providers are blocked by the company network.
3. The selected model is available on your plan.
4. Check the backend terminal for a more specific error.

---

## 8. Connect the helper

The helper is the optional program that reads git repositories on your computer. It also builds the
local code index that Copilot uses for code-aware answers.

### 8.1 Start the helper

Open a third terminal. Do not stop the backend or the website.

```bash
cd ~/Code/lunar/backend
go run ./cmd/lunar-helper
```

You should see:

```text
lunar helper listening address=127.0.0.1:5199 token_path=/Users/you/.lunar/helper-token
```

The exact home folder depends on your user account. Leave this terminal open.

On the first run, the helper creates:

| Item | Where |
|---|---|
| Helper token | `~/.lunar/helper-token` |
| Code index database | `~/.lunar/index/index.db` |

On Windows, those paths are under `%USERPROFILE%\.lunar\`.

On macOS and Linux, the token file is readable only by your user account. Treat it like a password.

### 8.2 Connect the helper in Lunar

1. Open **Settings**.
2. Open the **Workspace** tab.
3. Choose **On this computer**. This is the recommended choice.
4. Leave **Helper URL** as `http://127.0.0.1:5199`, unless you changed the helper port.
5. Click **Connect helper**. The browser fetches the token from the helper automatically.
6. Click **Test connection**. The status should become **Online**.
7. Enter your **Repository root folder**. This is the folder that contains your repository folders,
   for example `/Users/you/repositories`.
8. Click **Validate folder**.
9. Click **Save workspace**.

If automatic connection does not work, copy the token yourself:

```bash
cat ~/.lunar/helper-token
```

Paste it into **Helper token**, click **Test connection**, then click **Save workspace**. The pasted
token is kept for the current browser session and also stored for your account.

### 8.3 Choose the right repository root

The repository root is the **parent** folder, not one repository.

Good example:

```text
/Users/you/repositories
    service-a
    service-b
    service-c
```

Bad example:

```text
/Users/you/repositories/service-a
```

Lunar lists the folders inside the root and treats the ones that are git repositories as services.

### 8.4 Allow Lunar to change repositories

Reading is allowed by default. Actions that change a repository are blocked until you explicitly
allow the folder.

Actions that need the allowlist:

- Pull
- Fetch
- Stash changes local
- Discard changes
- Switch branch
- Create branches from a Jira issue
- Unit test runs and SOP scans from Confluence

Edit `backend/.env` and set:

```text
LUNAR_WORKSPACE_ALLOWED_ROOTS=/Users/you/repositories
```

You can allow more than one folder by separating them with commas:

```text
LUNAR_WORKSPACE_ALLOWED_ROOTS=/Users/you/repositories,/Users/you/other-repositories
```

Then restart both the backend and the helper. They both read this value.

If the list is empty, reads still work and writes are refused with a message about the allowlist. That
is intentional. It stops a stolen session from pointing Lunar at any folder and changing your work.

### 8.5 Choose repositories

1. Open **Microservices**.
2. Click the gear button, **Choose repositories**.
3. Tick the repositories you want to see.
4. Apply the selection.

The selection is per user. The board, the code index, and Copilot use it.

### 8.6 Build the code index

The index is a local searchable copy of your code. It lets Copilot find the right files quickly.

1. Open **Copilot**.
2. Click the code index chip in the top header.
3. Click **Build index**.
4. Wait until the status becomes ready.

The chip shows the number of repositories, files, chunks, and the last update time.

The current build is incremental. Files that did not change are not re-read.

The index database stays on your computer. When you ask a code question, the helper reads the
relevant pieces and sends them to the selected AI provider together with your question. The backend
does not store the index.

### 8.7 Helper settings

The helper reads these values from `backend/.env`:

| Name | Default | Meaning |
|---|---|---|
| `LUNAR_HELPER_ADDR` | `127.0.0.1:5199` | Listen address. A non-loopback host is forced back to `127.0.0.1` |
| `LUNAR_HELPER_ORIGINS` | `http://localhost:5174`, `http://127.0.0.1:5174` | Browser origins allowed to call the helper |
| `LUNAR_WORKSPACE_ALLOWED_ROOTS` | empty | Folders that may be modified |

If your frontend runs on a different address, add that exact address to `LUNAR_HELPER_ORIGINS` and
restart the helper.

### 8.8 Build a standalone helper binary

`go run` is fine for daily use. If you want a binary you can keep:

```bash
cd ~/Code/lunar/backend
mkdir -p ~/bin
go build -o ~/bin/lunar-helper ./cmd/lunar-helper
~/bin/lunar-helper
```

Run it from the `backend` folder so it can read `backend/.env`.

---

## 9. A tour of every page

This section explains what each page does and what to click. Setup must be finished first.

### 9.1 Dashboard

The Dashboard is the landing page after sign-in.

What you see:

- **Jira Assigned Tickets**: tickets assigned to you.
- **Bitbucket Pull Requests**: pull requests that involve you.
- **Technical Documents**: documents from Confluence.
- **AI Providers**: how many providers are configured.

What you can do:

- Click **Sync workspace** to refresh the numbers. The page shows the last sync time.
- Use the quick access tiles to jump to Copilot, Jira, Bitbucket, or Settings.
- Read the integration status to see which system still needs a token.

If one system is down, the other cards still show their own data. A red or warning card is not a
reason to panic; open the matching page to see the detailed error.

### 9.2 Jira

Open **Jira**.

What you see:

- **Assigned to me**: your issues on a board, grouped by status.
- **Backlog**: the active sprint of your own squad, grouped by sprint. Sprint groups open and close.

Filters on the assigned board:

- **All**
- **DEV Documents**, with a second filter for **UT**, **Query Review**, or **SOP**
- **Bug / Defect**
- **Subtask**

What you can do:

- Click a card to open the issue detail.
- Read the fields, remote links, and the link to the issue on BRI Jira.
- Use **Refresh** to fetch the latest issues.
- Open the branches panel to see branches whose names contain the issue key.
- Create a branch in one or more repositories. Choose the base branch for each repository. Lunar
  reports each repository separately, so a partial result is visible.
- Use the repository suggestion to see which repositories the issue will probably touch. A low
  confidence result is a list of candidates, not a command.

If Jira returns a `502` and mentions connectivity, turn on VPN and try **Refresh** again. Lunar never
shows invented tickets when Jira is unreachable.

If the backlog is empty, Lunar could not detect your squad or your squad has no active sprint. It does
not show another squad's work.

### 9.3 Bitbucket

Open **Bitbucket**.

The page has two main areas.

**Your work** shows branches you pushed recently.

- Filter by **All**, **Ready**, or **Stale**.
- Click **Create pull request** on a branch.
- Fill in the pull request form and submit.

**Pull requests** shows open pull requests.

- Filter by **All**, **Assigned to me**, or **AI flagged**.
- Click a pull request to open the review dialog.

Inside the review dialog:

1. The left side shows the changed files. Click a file to open or close it.
2. The right side shows findings and the review comment.
3. Choose a model in the model picker. This model is independent of the Copilot chat model.
4. Click **Review with AI**.
5. Wait for the answer. Findings are grouped into blocker, suggestion, and nitpick.
6. Read the assumptions section. A finding that is not present in the diff is marked as an assumption,
   not as a fact.
7. Edit the comment if you want, or preview it as Markdown.
8. Post the comment, approve, mark it as needs work, or decline.
9. Decline asks for confirmation first, because the author sees it immediately.

If no AI provider is configured, the dialog shows a banner and a link to Settings. If the review fails,
use **Try again**. Lunar does not replace a failed review with a canned answer.

### 9.4 Confluence

Open **Confluence**.

The list page shows every technical document, with counts for:

- **All documents**
- **UT Documents**
- **Query Review**
- **SOP**

You can filter by type and search by text.

Click a document to open it. The detail page shows the body, metadata, status, labels, and comments.

There are three AI actions, depending on the document:

**Extract query review**

- Finds the sibling `<KEY>-MIG` page under the same parent.
- Reads its repository, branch, and compare-link table.
- Reads the branch comparison and extracts the queries introduced or changed by added lines.
- Skips `fe-mms` and `mms`.
- Shows one result per repository, with analyzed files, skipped files, queries, and a truncation flag.
- Renders the result as the `Capture explain (mysql)` table.
- **Copy table** copies rich HTML, so it pastes into Confluence as a real table.
- **Instant apply** writes the latest table back to the page and increases the page version.

**Generate UT Docs With AI**

- Reads the sibling `<KEY>-MIG` page to find the repositories and branches to run.
- Uses the helper to check out each branch, stash a dirty worktree, scan the changed test files, and
  run only the safe mock-based Go tests.
- Blocks test files that can reset, refresh, flush, or drop a database, or open a real connection.
- Streams the phases and output per repository.
- Renders the `Normal Test Scenario` table and a run screenshot.
- **Instant apply** uploads the screenshots and writes the rows to the `<KEY>-UT` page.

**Generate SOP Deploy & Operasional**

- Appears on a document whose title carries an issue key.
- Finds `<KEY>-Development` and reads the sibling `<KEY>-MIG` repositories.
- Scans changed SQL and DDL files through the helper.
- Builds `<KEY>-SOP Deploy` with database, application, rollback, and database rollback chapters.
- Builds the `<KEY>-SOP Operasional` plan with maintenance, monitoring, and troubleshooting.
- SQL statements come from the repository files. The model does not invent SQL.
- **Instant apply** creates or updates the pages. Only the database chapters are managed
  automatically, so manual screenshots in the application chapters are kept.

Requirements for these actions:

- Confluence and Bitbucket tokens, depending on the action.
- An AI provider key.
- A running helper, for the actions that read repositories or run tests.
- `LUNAR_WORKSPACE_ALLOWED_ROOTS` containing the workspace root, for the actions that modify a
  worktree or run tests.

**Instant apply writes to Confluence.** Read the preview before you use it.

### 9.5 Microservices

Open **Microservices**.

This page shows the services in the repositories you selected.

What you see:

- A search box for a service or branch.
- Filters: **All**, **Dirty**, **Clean**. Dirty means the repository has changed files.
- A service table with the current branch, last commit, changed file count, and an action menu.

What you can do:

- Click **Choose repositories** (the gear button) to change the list.
- Click a service row to open its detail panel. It shows the branch, last commit, changed files, and
  file list.
- Click the branch name to open the branch picker. Choosing a branch runs `git checkout`.
- Use the kebab menu for **Pull**, **Fetch**, **Stash changes local**, **Discard changes**, and
  **Details**. **Discard changes** asks for confirmation and restores tracked files only.
- In the detail panel, click **Choose AI model** to export the service as a Postman collection.

The Postman export:

1. Reads the repository source.
2. Builds a prompt for the selected model.
3. Shows the stages: selecting, prompt, model, validating.
4. Produces a Postman Collection v2.1.
5. Shows the request and folder counts.
6. Downloads `<service>-postman-collection.json` in your browser.

Nothing is written on the server. The export only reads the repository.

Requirements:

- An AI provider key.
- A repository source. Use the helper for repositories on your computer.
- The write allowlist for Pull, Fetch, Stash, Discard, and branch switching.

If the helper is offline, the page shows a setup guide and a **Check again** control.

### 9.6 Copilot

Open **Copilot**.

**Start a chat**

- Click **New chat** in the sidebar, click the `+` button in the header, or press `Command + Shift + O`
  on macOS or `Ctrl + Shift + O` on Windows and Linux.
- Type a question in the box.
- Press `Enter` to send. Press `Shift + Enter` to add a new line.
- The answer streams in as it is generated.

**Read an answer**

- The assistant answer can contain Markdown, code blocks, tables, and Mermaid diagrams.
- Code blocks have a copy button, a language label, and line numbers.
- Diagrams render in place under the heading where the model wrote them.
- The reasoning panel opens when the model returns reasoning.
- The tool trail shows which tools the agent used.
- Use **Copy** to copy an answer and **Regenerate** to ask again.

**Choose a model**

- Click the model trigger in the composer.
- The first level offers **Model** and **Effort**.
- In **Model**, search for a name and choose a configured model.
- In **Effort**, choose Low, Med, or High.
- Thinking mode is always on. The effort is remembered across reloads, but some providers may ignore
  it.

**Use code context**

- The code index chip is in the Copilot header.
- The chip shows whether the index is missing, building, or ready.
- Click it to see repository, file, and chunk counts, then click **Build index**.
- With the helper connected and the index ready, Copilot can search code, read files, find symbol
  references, list services, read routes, and inspect git diffs.
- Without the helper or the index, Copilot still answers, but the answer has no code context and the
  page says why.

**Choose tool domains**

- Click the tool settings button in the chat sidebar.
- The **Active Copilot Tools** dialog opens.
- Choose the domains you want, then close the dialog.
- More tools make the prompt longer and cost more tokens.
- The Jira tools work. The Bitbucket and Confluence tools are not implemented yet.

**Attachments**

The backend supports attachments, but the composer control is not built yet. Do not expect a paperclip
button in this version.

**Conversation history**

Conversations are saved on the backend, so they survive a helper restart. Deleting a conversation asks
for confirmation first.

### 9.7 Settings

Settings has four tabs:

| Tab | What it holds |
|---|---|
| Integrations | Jira, Bitbucket, and Confluence tokens |
| Workspace | Repository source, helper connection, and root folder |
| AI providers | Keys for the six AI providers |
| Security | Encryption status |

The Security tab confirms that secrets are encrypted at rest with AES-256-GCM.

---

## 10. Stop, restart, update, and reset

### 10.1 Stop everything

In each terminal, press `Control + C`.

You have three terminals:

1. Backend
2. Frontend
3. Helper

Closing a terminal also stops the program inside it.

### 10.2 Start again later

Open three terminals and run:

```bash
cd ~/Code/lunar/backend
go run ./cmd/server
```

```bash
cd ~/Code/lunar/frontend
npm run dev
```

```bash
cd ~/Code/lunar/backend
go run ./cmd/lunar-helper
```

The order is not strict. The helper is independent of the backend.

Your data stays:

- Accounts, settings, encrypted tokens, and chat history are in `backend/data/lunar.db`.
- The helper token and code index are in `~/.lunar/`.

### 10.3 Update the code

Stop the three programs first, then run:

```bash
cd ~/Code/lunar
git pull
cd backend
go mod download
cd ../frontend
npm install
```

Start them again.

### 10.4 Reset the backend data

This deletes every account, saved token, workspace setting, and chat history in the local database.

```bash
cd ~/Code/lunar/backend
rm -f data/lunar.db data/lunar.db-wal data/lunar.db-shm
```

Start the backend again. It recreates the database and the first user from `.env`.

Make a backup first if you are not sure:

```bash
cp data/lunar.db data/lunar.db.backup
```

### 10.5 Reset the helper

This deletes the helper token and the code index.

```bash
rm -rf ~/.lunar
```

Start the helper again. It creates a new token and an empty index. Reconnect the helper in Settings and
build the index again.

---

## 11. Problems and fixes

Read the first error message in the terminal. It usually names the exact problem.

| Symptom | Likely cause | Fix |
|---|---|---|
| `command not found: go` | Go is not installed or not on the PATH | Install Go, open a new terminal, run `go version` |
| `command not found: node` or `npm` | Node.js is not installed or not on the PATH | Install Node.js LTS, open a new terminal |
| `Unsupported engine` or a Vite Node version error | Node.js is too old | Install Node.js 22.18+ |
| Backend says `address already in use` | Another program uses port `8081` | Stop that program, or change `PORT` and the Vite proxy target |
| Backend exits immediately | Bad `.env`, unwritable database path, or an empty encryption key | Read the first log line; check `ENCRYPTION_KEY` is not empty and the database path is writable |
| Website is blank or every API call fails | Backend is not running, or the proxy target is wrong | Start the backend and open `http://localhost:8081/api/health` |
| Vite starts on `5175` | Port `5174` is busy | Stop the other program, or add the new origin to `LUNAR_HELPER_ORIGINS` |
| Cannot sign in | Wrong email or password, or the session expired | Use the seed account or create an account; sign in again |
| Seed account does not work | `.env` has different `SEED_USER_*` values, or the database already had that user | Check `.env`; the seed only creates a user when the email is absent |
| Register fails at Verify Identity | Bad Jira PAT, wrong Jira URL, or VPN is off | Check the token, the base URL, and VPN |
| Register says the account already exists | The BRI identity is already registered | Sign in instead |
| Register password is rejected | Password is shorter than 6 characters | Use at least 6 characters |
| Jira shows `502` or a VPN hint | Backend cannot reach BRI Jira | Turn on VPN, check the PAT, try Refresh |
| Jira backlog is empty | No active sprint, or Lunar could not detect your squad | Check that you have issues in an active sprint |
| Bitbucket returns `401` or `403` | Wrong username or PAT, or missing permission | Re-enter the username and PAT in Settings |
| Confluence returns `401` or `403` | Wrong PAT or missing permission | Re-enter the PAT in Settings |
| Copilot model list is empty | No AI provider key is configured | Add a key in Settings, or ask for a shared system key |
| AI request fails | Wrong key, no credit, blocked network, or unavailable model | Check the backend log and the provider account |
| Helper shows Offline | Helper is not running, or the URL is wrong | Start the helper and check `http://127.0.0.1:5199/health` |
| Helper returns `401 unauthorized` | Token is missing or wrong | Click **Connect helper** again, or paste `~/.lunar/helper-token` |
| Helper returns `403` in the browser console | The page origin is not allowed | Use `localhost:5174` or `127.0.0.1:5174`, or update `LUNAR_HELPER_ORIGINS` and restart the helper |
| `workspace root does not exist` | The folder is not on this computer, or the path is wrong | Use a folder that exists on this computer |
| A folder is listed as `not a git repository` | The folder is a plain folder | This is expected; choose a real repository |
| Pull, Fetch, Stash, Discard, or branch switching is refused | `LUNAR_WORKSPACE_ALLOWED_ROOTS` is empty or does not contain the root | Set it in `backend/.env` and restart the backend and helper |
| Branch switching is refused | The repository has tracked changes | Stash or discard the changes first |
| Code index is missing | Repositories are not selected, the root is empty, or the helper is offline | Select repositories, set the root, connect the helper, then Build index |
| Unit test documents or SOP cannot run | Helper is offline or the allowlist is missing | Start the helper and set `LUNAR_WORKSPACE_ALLOWED_ROOTS` |
| Postman export is locked | No AI provider key, or no repository source | Add an AI key, then connect the helper or choose the server source |
| E2E tests fail | The app is not running, Playwright browsers are missing, or external services are unreachable | Start backend and frontend, run `npx playwright install`, check VPN |
| Changes to `.env` do nothing | The program was not restarted, or you edited a different `.env` | Restart the program; edit `backend/.env` |
| Frontend cannot reach the helper from another computer | The helper only listens on loopback | This is by design; run Lunar and the helper on the same computer |

### 11.1 Quick health checks

Backend:

```bash
curl http://localhost:8081/api/health
```

Helper:

```bash
curl http://127.0.0.1:5199/health
```

Helper repositories, with the token:

```bash
curl -H "Authorization: Bearer $(cat ~/.lunar/helper-token)" "http://127.0.0.1:5199/repositories?root=/Users/you/repositories"
```

Ports in use on macOS or Linux:

```bash
lsof -i :8081
lsof -i :5174
lsof -i :5199
```

### 11.2 Read the terminal

When something fails, the terminal that runs the failing piece usually has the useful message:

- Backend errors appear in the backend terminal.
- Website build errors appear in the frontend terminal.
- Git and index errors appear in the helper terminal.

Copy the first line that contains `error`, not the whole screen.

---

## 12. For developers

### 12.1 Backend tests

```bash
cd ~/Code/lunar/backend
go test ./...
```

Race detector:

```bash
go test -race ./...
```

Static checks:

```bash
go vet ./...
```

Build the backend:

```bash
go build ./...
```

### 12.2 Frontend tests

```bash
cd ~/Code/lunar/frontend
npm test
```

Production type check and build:

```bash
npm run build
```

End-to-end tests:

```bash
npx playwright install chromium
npm run test:e2e
```

The unit test command runs TypeScript files directly, so use Node.js 22.18 or newer for that command.

The end-to-end tests expect the backend and frontend to be running. They sign in with the seeded user
or with `E2E_USER_EMAIL` and `E2E_USER_PASSWORD`.

Run one area at a time when you are fixing a specific feature:

```bash
npm run test:e2e:auth
npm run test:e2e:dashboard
npm run test:e2e:copilot
npm run test:e2e:jira
npm run test:e2e:bitbucket
npm run test:e2e:confluence
npm run test:e2e:settings
```

Some end-to-end tests need real BRI services. A failure there may be a network or credential problem,
not a code problem.

### 12.3 Where the code lives

| Folder | What it contains |
|---|---|
| `backend/cmd/server` | The backend program |
| `backend/cmd/lunar-helper` | The helper program |
| `backend/cmd/seed` | A command that creates a user directly in the database |
| `backend/internal` | Backend features: auth, Jira, Bitbucket, Confluence, Copilot, workspace, postman |
| `frontend/src` | Vue application |
| `frontend/src/features` | One folder per page: dashboard, jira, bitbucket, confluence, copilot, microservices, settings |
| `frontend/e2e` | Playwright tests |
| `docs/microservices/HELPER.md` | Helper installation, endpoints, and troubleshooting |
| `guide.md` | The full feature inventory and change log |
| `plan/` | Feature plans and the backlog |
| `deploy/` | systemd and nginx files for a server deployment |

### 12.4 Create a user directly

The server normally creates the first user from `.env`. You can also create one with the seed command:

```bash
cd ~/Code/lunar/backend
go run ./cmd/seed you@example.com your-password "Your Name"
```

This writes to `DB_PATH`, which defaults to `data/lunar.db`. The command stops if the email already
exists.

### 12.5 Production deployment

The repository includes:

- `deploy/lunar-backend.service` for systemd.
- `deploy/nginx.conf` for nginx.

Before a real deployment:

1. Set a strong `JWT_SECRET`.
2. Set a strong `ENCRYPTION_KEY` and keep it stable.
3. Set `CORS_ORIGIN` to the real frontend address, not `*`.
4. Point `JIRA_BASE_URL`, `BITBUCKET_BASE_URL`, and `CONFLUENCE_BASE_URL` at the right systems.
5. Build the frontend with `npm run build` and serve the generated files.
6. Keep `backend/data/lunar.db` on a persistent volume and back it up.
7. Do not run the helper on a shared server. It belongs on the developer's own computer.
8. Keep the helper on loopback and keep its origin allowlist narrow.
9. Change or remove the default seeded account before other people can reach the deployment.

### 12.6 Not implemented yet

Check `guide.md`, section 11, before you promise a feature. Known gaps include:

- The attachment picker in the Copilot composer.
- Bitbucket and Confluence agent tools.
- Semantic code search using embeddings.
- Reasoning effort reaching every provider.
- A helper download page.

---

## 13. Safety rules

These rules protect your accounts and your code.

1. **Never commit `.env`.** It contains secrets.
2. **Never paste tokens into a chat, a ticket, or a screenshot.**
3. **Keep `ENCRYPTION_KEY` stable.** Changing it makes saved tokens unreadable.
4. **Use a strong `JWT_SECRET` and a narrow `CORS_ORIGIN` before sharing a deployment.**
5. **Run the helper only on your own computer.** It holds source code.
6. **Keep the helper on `127.0.0.1`.** Do not expose port `5199` to a network.
7. **Set `LUNAR_WORKSPACE_ALLOWED_ROOTS` deliberately.** It is the list of folders Lunar may modify.
8. **Read the preview before Instant apply.** It writes to Confluence.
9. **Read the confirmation before Discard.** It cannot be undone.
10. **Read the confirmation before Decline.** The author sees it immediately.
11. **Back up `backend/data/lunar.db`.** It holds users, tokens, settings, and chat history.
12. **Do not delete `~/.lunar` unless you mean to reset the helper.** It removes the token and index.
13. **Log out on a shared computer.**

---

## 14. Glossary

| Word | Simple meaning |
|---|---|
| API | A way for two programs to talk to each other |
| API key | A secret password for an AI provider |
| Backend | The program that holds secrets and talks to company systems |
| Branch | A separate line of work in a git repository |
| BRI | Bank Rakyat Indonesia, the company this app is built for |
| CORS | A browser rule that decides which website may call a local service |
| Diff | The list of lines that changed between two versions of code |
| Dirty | A repository with changed files that are not committed |
| Frontend | The website you see |
| Git | The tool that tracks changes in code |
| Helper | The small program that reads repositories on your own computer |
| Index | A local searchable copy of code used by Copilot |
| JWT | The login token the browser uses after sign-in |
| MIG page | A Confluence page named `<KEY>-MIG` that lists repositories and branches for an issue |
| PAT | Personal Access Token, a secret password for Jira, Bitbucket, or Confluence |
| Port | A numbered door on a computer, such as `8081` |
| PR | Pull request, a request to merge a branch |
| Repository | A folder whose history is tracked by git |
| SOP | Standard Operating Procedure, a step-by-step operations document |
| Sprint | A fixed period of work in Jira |
| SQL | The language used to talk to a database |
| Terminal | The window where you type commands |
| Token | A secret string that proves who you are or lets a program in |
| UT | Unit Test, a small test for one piece of code |
| VPN | A secure connection to the company network |
| Workspace root | The parent folder that contains your repository folders |

---

## 15. Learn more

| Document | What it covers |
|---|---|
| `guide.md` | Every feature and subfeature, plus the change log |
| `docs/microservices/HELPER.md` | Helper installation, endpoints, and troubleshooting |
| `AGENTS.md` | Engineering rules for people and AI agents working on this repository |
| `docs/qa/README.md` | QA notes and reports |
| `plan/BACKLOG.md` | Open work that is not finished |
| `plan/<feature>/` | Plans for individual features |

If you are new, start with this README. If you need the exact behaviour of a feature, read `guide.md`.
If the helper does not work, read `docs/microservices/HELPER.md`.
