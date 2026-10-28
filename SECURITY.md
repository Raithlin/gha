# Security policy

## Supported versions

GHA is prerelease software. Security fixes target the latest published release,
including prereleases, and the current development branch. Older versions may
require an upgrade rather than a backported fix.

If you find a vulnerability in an older version, please report it even if you
cannot test the latest release.

## Report a vulnerability privately

Use GitHub's [Report a vulnerability](https://github.com/Raithlin/gha/security/advisories/new)
form. You can also find it under **Security → Advisories** in this repository.
Reports go privately to the maintainer, Stephen (@Raithlin).

Please avoid public issues, discussions, or pull requests that disclose an
unfixed vulnerability.

Include what you can:

- The affected GHA version, installation method, and operating system.
- The command or workflow involved and any access needed to reproduce the issue.
- Reproduction steps or a small proof of concept using a disposable repository.
- The security impact and any workaround you have found.

Remove tokens, passwords, private repository content, and personal data from
reports and attachments. Use test credentials and synthetic examples.

Reports may cover the CLI, agent setup, release packages, or project-maintained
installation and release workflows. Examples include credential exposure,
unintended command execution, or changes to files or repositories beyond the
requested target. If you are unsure whether a bug has a security impact,
report it privately and explain your concern.

## What happens next

The maintainer will review the report, ask for details where needed, and
coordinate a fix and disclosure with the reporter. This project does not
promise a fixed response or resolution time. Please allow time to investigate
and coordinate disclosure before publishing details.

For ordinary bugs, feature requests, or help using GHA, follow the
[contribution guide](docs/CONTRIBUTING.md).
