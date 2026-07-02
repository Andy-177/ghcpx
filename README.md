# ghcp [![go](https://github.com/int128/ghcp/actions/workflows/go.yaml/badge.svg)](https://github.com/int128/ghcp/actions/workflows/go.yaml)

This is a release engineering tool for GitHub.
It depends on GitHub APIs and works without git installation.

It provides the following features:

- Commit files to a repository
- Create an empty commit
- Fork a repository and commit files to the forked repository
- Create a pull request
- Upload files to GitHub Releases


## Getting Started

### 1. Install

You can install the latest release from [GitHub Releases](https://github.com/int128/ghcp/releases) or Homebrew.

```sh
# GitHub Releases
curl -fL -o /tmp/ghcp.zip https://github.com/int128/ghcp/releases/download/v1.8.0/ghcp_linux_amd64.zip
unzip /tmp/ghcp.zip -d ~/bin

# Homebrew
brew install int128/ghcp/ghcp
```

### 2. Set up password

Run the following command to set up a password:

```sh
ghcp passwd set <password>
```

This creates `ghcp.json` in the same directory as the binary, containing the SHA256 hash of the password.

### 3. Store GitHub Token

Get a personal access token from the [settings](https://github.com/settings/tokens) and store it with the password:

```sh
ghcp <password> token <github_token>
```

The token is encrypted with AES-256-CBC and stored in `ghcp.json`.

### 4. Use

All commands require the password as the first argument:

```sh
ghcp <password> commit -r OWNER/REPO -m MESSAGE file1 file2
```

The password is used to decrypt the GitHub Token from `ghcp.json`.


## Password Management

### Set password

```sh
ghcp passwd set <password>
```

### Change password

Decrypts the token with the old password, re-encrypts with the new one:

```sh
ghcp passwd change <old_password> <new_password>
```

### Store token

Encrypts and stores the GitHub Token using the password:

```sh
ghcp <password> token <github_token>
```

### `--token` override

You can still use the `--token` flag to bypass the password system:

```sh
ghcp commit --token <token> -r OWNER/REPO -m MESSAGE file1 file2
```


### Commit files to a branch

To commit files to the default branch:

```sh
ghcp <password> commit -r OWNER/REPO -m MESSAGE file1 file2
```

To commit files to `feature` branch:

```sh
ghcp <password> commit -r OWNER/REPO -b feature -m MESSAGE file1 file2
```

If `feature` branch does not exist, ghcp will create it from the default branch.

To create `feature` branch from `develop` branch:

```sh
ghcp <password> commit -r OWNER/REPO -b feature --parent=develop -m MESSAGE file1 file2
```

If `feature` branch already exists, ghcp will fail.
Currently only fast-forward is supported.

ghcp performs a commit operation as follows:

- An author and committer of a commit are set to the login user (depending on the token).
- If the branch has same files, do not create a new commit. It prevents an empty commit.
- It excludes `.git` directories.
- It does not support `.gitconfig`.

You can set the following options.

```
Flags:
      --author-email string      Author email (default: login email)
      --author-name string       Author name (default: login name)
  -b, --branch string            Name of the branch to create or update (default: the default branch of repository)
      --committer-email string   Committer email (default: login email)
      --committer-name string    Committer name (default: login name)
      --dry-run                  Upload files but do not update the branch actually
  -h, --help                     help for commit
  -m, --message string           Commit message (mandatory)
      --no-file-mode             Ignore executable bit of file and treat as 0644
      --no-parent                Create a commit without a parent
  -u, --owner string             Repository owner
      --parent string            Create a commit from the parent branch/tag (default: fast-forward)
  -r, --repo string              Repository name, either -r OWNER/REPO or -u OWNER -r REPO (mandatory)
```


### Create an empty commit to a branch

To create an empty commit to the default branch:

```sh
ghcp <password> empty-commit -r OWNER/REPO -m MESSAGE
```

To create an empty commit to the branch:

```sh
ghcp <password> empty-commit -r OWNER/REPO -b BRANCH -m MESSAGE
```

If the branch does not exist, ghcp creates a branch from the default branch.
It the branch exists, ghcp updates the branch by fast-forward.

To create an empty commit to a new branch from the parent branch:

```sh
ghcp <password> empty-commit -r OWNER/REPO -b BRANCH --parent PARENT -m MESSAGE
```

If the branch exists, it will fail.

You can set the following options.

```
Flags:
      --author-email string      Author email (default: login email)
      --author-name string       Author name (default: login name)
  -b, --branch string            Name of the branch to create or update (default: the default branch of repository)
      --committer-email string   Committer email (default: login email)
      --committer-name string    Committer name (default: login name)
      --dry-run                  Do not update the branch actually
  -h, --help                     help for empty-commit
  -m, --message string           Commit message (mandatory)
  -u, --owner string             Repository owner
      --parent string            Create a commit from the parent branch/tag (default: fast-forward)
  -r, --repo string              Repository name, either -r OWNER/REPO or -u OWNER -r REPO (mandatory)
```


### Fork the repository and commit files to a new branch

To fork repository `UPSTREAM/REPO` and create `feature` branch from the default branch:

```sh
ghcp <password> fork-commit -u UPSTREAM/REPO -b feature -m MESSAGE file1 file2
```

To fork repository `UPSTREAM/REPO` and create `feature` branch from `develop` branch of the upstream:

```sh
ghcp <password> fork-commit -u UPSTREAM/REPO -b feature --parent develop -m MESSAGE file1 file2
```

If the branch already exists, ghcp will fail.
Currently only fast-forward is supported.

You can set the following options.

```
Flags:
      --author-email string      Author email (default: login email)
      --author-name string       Author name (default: login name)
  -b, --branch string            Name of the branch to create (mandatory)
      --committer-email string   Committer email (default: login email)
      --committer-name string    Committer name (default: login name)
      --dry-run                  Upload files but do not update the branch actually
  -h, --help                     help for fork-commit
  -m, --message string           Commit message (mandatory)
      --no-file-mode             Ignore executable bit of file and treat as 0644
  -u, --owner string             Upstream repository owner
      --parent string            Upstream branch name (default: the default branch of the upstream repository)
  -r, --repo string              Upstream repository name, either -r OWNER/REPO or -u OWNER -r REPO (mandatory)
```


### Create a pull request

To create a pull request from `feature` branch to the default branch:

```sh
ghcp <password> pull-request -r OWNER/REPO -b feature --title TITLE --body BODY
```

To create a pull request from `feature` branch to the `develop` branch:

```sh
ghcp <password> pull-request -r OWNER/REPO -b feature --base develop --title TITLE --body BODY
```

To create a pull request from `feature` branch of `OWNER/REPO` repository to the default branch of `UPSTREAM/REPO` repository:

```sh
ghcp <password> pull-request -r OWNER/REPO -b feature --base-repo UPSTREAM/REPO --title TITLE --body BODY
```

To create a pull request from `feature` branch of `OWNER/REPO` repository to the default branch of `UPSTREAM/REPO` repository:

```sh
ghcp <password> pull-request -r OWNER/REPO -b feature --base-repo UPSTREAM/REPO --base feature --title TITLE --body BODY
```

If an open pull request already exists, ghcp does nothing.

You can set the following options.

```
Flags:
      --base string         Base branch name (default: default branch of base repository)
      --base-owner string   Base repository owner (default: head)
      --base-repo string    Base repository name, either --base-repo OWNER/REPO or --base-owner OWNER --base-repo REPO (default: head)
      --body string         Body of a pull request
      --draft               If set, mark as a draft
  -b, --head string         Head branch name (mandatory)
  -u, --head-owner string   Head repository owner
  -r, --head-repo string    Head repository name, either -r OWNER/REPO or -u OWNER -r REPO (mandatory)
  -h, --help                help for pull-request
      --reviewer string     If set, request a review
      --title string        Title of a pull request (mandatory)
```


### Release assets

To upload files to the release associated to tag `v1.0.0`:

```sh
ghcp <password> release -r OWNER/REPO -t v1.0.0 dist/
```

If the release does not exist, it will create a release.
If the tag does not exist, it will create a tag from the default branch and create a release.

To create a tag and release on commit `COMMIT_SHA` and upload files to the release:

```sh
ghcp <password> release -r OWNER/REPO -t v1.0.0 --target COMMIT_SHA dist/
```

If the tag already exists, it ignores the target commit.
If the release already exist, it only uploads the files.

You can set the following options.

```
Flags:
      --dry-run         Do not create a release and assets actually
  -h, --help            help for release
  -u, --owner string    Repository owner
  -r, --repo string     Repository name, either -r OWNER/REPO or -u OWNER -r REPO (mandatory)
  -t, --tag string      Tag name (mandatory)
      --target string   Branch name or commit SHA of a tag. Unused if the Git tag already exists (default: the default branch)
```


## Usage

### Global options

You can set the following options.

```
Global Flags:
      --api string         GitHub API v3 URL (v4 will be inferred) [$GITHUB_API]
      --debug              Show debug logs
  -C, --directory string   Change to directory before operation
      --token string       GitHub API token (bypasses password)
```

### GitHub Enterprise

You can set a GitHub API v3 URL by `GITHUB_API` environment variable or `--api` option.

```sh
export GITHUB_API=https://github.example.com/api/v3/
```

GitHub API v4 URL will be automatically inferred from the v3 URL by resolving the relative path `../graphql`.


### Config file

`ghcp.json` is stored in the same directory as the binary:

```json
{
  "password": "<sha256 hash>",
  "github_token": "<aes-256-cbc encrypted token>"
}
```


## Contributions

This is an open source software.
Feel free to open issues and pull requests.

Author: [Hidetake Iwata](https://github.com/int128)
