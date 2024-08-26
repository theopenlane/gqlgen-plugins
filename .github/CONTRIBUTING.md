# Contributing

Given external users will not have write to the branches in this repository, you'll need to follow the forking process to open a PR - [here](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/creating-a-pull-request-from-a-fork) is a guide from github on how to do so.

Please also read our main [contributing guide](https://github.com/theopenlane/.github/blob/main/CONTRIBUTING.md) in addition to this one; the main guide mostly says that we'd like for you to open an issue first but it's not hard-required, and that we accept all forms of proposed changes given the state of this code base (in it's infancy, still!)

## Pre-requisites to a PR

This repository contains a number of code generating functions / utilities which take schema modifications and scaffold out resolvers, graphql API schemas, openAPI specifications, among other things. To ensure you've generated all the necessary dependencies run `task pr`; this will run the entirety of the commands required to safely generate a PR. If for some reason one of the commands fails / encounters an error, you will need to debug the individual steps. It should be decently easy to follow the `Taskfile` in the root of this repository.

### Pre-Commit Hooks

We have several `pre-commit` hooks that should be run before pushing a commit. Make sure this is installed:

```bash
brew install pre-commit
pre-commit install
```

You can optionally run against all files:

```bash
pre-commit run --all-files
```
