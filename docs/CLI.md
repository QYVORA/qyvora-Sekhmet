# CLI Reference

`sekhmet` follows the shared QYVORA tier grammar:

```
sekhmet <tier> <module> [<action>] [target] [flags]
```

`<tier>` is one of `recon`, `exploit`, or `report`. Every pre-tier command remains available as a hidden alias.

## recon

| Command | Purpose |
|---------|---------|
| `sekhmet recon analyze` | Classify and summarize the findings of a session |
| `sekhmet recon baseline` | Profile the target's normal behaviour before fuzzing |
| `sekhmet recon capabilities` | List sekhmet's machine-readable capabilities |
| `sekhmet recon completion` | Generate shell completion scripts |
| `sekhmet recon corpus` | Manage the seed corpus |
| `sekhmet recon crashes` | List deduplicated crashes from a session |
| `sekhmet recon fuzz` | Run a fuzzing campaign against the current target |
| `sekhmet recon help` | Help about any command |
| `sekhmet recon minimize` | Minimize an interesting input against the current target |
| `sekhmet recon replay` | Replay a single input against the current target |
| `sekhmet recon report` | Render a campaign report for a session |
| `sekhmet recon session` | Manage fuzzing sessions |
| `sekhmet recon target` | Manage fuzzing targets |
| `sekhmet recon tui` | start the interactive terminal application |
| `sekhmet recon updates` | Report version and update channel |
| `sekhmet recon version` | Print sekhmet version and build information |
| `sekhmet recon wordlists` | SecLists integration: list categories and search wordlists |

## exploit

| Command | Purpose |
|---------|---------|
| `sekhmet exploit analyze` | Classify and summarize the findings of a session |
| `sekhmet exploit baseline` | Profile the target's normal behaviour before fuzzing |
| `sekhmet exploit capabilities` | List sekhmet's machine-readable capabilities |
| `sekhmet exploit completion` | Generate shell completion scripts |
| `sekhmet exploit corpus` | Manage the seed corpus |
| `sekhmet exploit crashes` | List deduplicated crashes from a session |
| `sekhmet exploit fuzz` | Run a fuzzing campaign against the current target |
| `sekhmet exploit help` | Help about any command |
| `sekhmet exploit minimize` | Minimize an interesting input against the current target |
| `sekhmet exploit replay` | Replay a single input against the current target |
| `sekhmet exploit report` | Render a campaign report for a session |
| `sekhmet exploit session` | Manage fuzzing sessions |
| `sekhmet exploit target` | Manage fuzzing targets |
| `sekhmet exploit tui` | start the interactive terminal application |
| `sekhmet exploit updates` | Report version and update channel |
| `sekhmet exploit version` | Print sekhmet version and build information |
| `sekhmet exploit wordlists` | SecLists integration: list categories and search wordlists |

## report

_No report-tier modules._

## Utilities

| Command | Purpose |
|---------|---------|
| `sekhmet analyze` | Classify and summarize the findings of a session |
| `sekhmet baseline` | Profile the target's normal behaviour before fuzzing |
| `sekhmet corpus` | Manage the seed corpus |
| `sekhmet crashes` | List deduplicated crashes from a session |
| `sekhmet fuzz` | Run a fuzzing campaign against the current target |
| `sekhmet minimize` | Minimize an interesting input against the current target |
| `sekhmet replay` | Replay a single input against the current target |
| `sekhmet session` | Manage fuzzing sessions |
| `sekhmet target` | Manage fuzzing targets |
| `sekhmet tui` | start the interactive terminal application |
| `sekhmet wordlists` | SecLists integration: list categories and search wordlists |
| `docs script <file>` | Run a `.qs` script with the shared scripting engine (`run`/`build` subcommands) |
