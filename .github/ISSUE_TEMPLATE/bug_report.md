---
name: Bug Report
about: Report a bug or unexpected behavior
title: "[BUG] "
labels: bug
---

## Description

<!-- A clear and concise description of what the bug is -->

## Kit

<!-- cloudsmith-repo, cloudsmith-cli or cloudsmith-dependency-firewall -->

## Steps to Reproduce

<!-- Include the full `sbx run` or `sbx create` command, with every --kit and
     --kit-arg. Replace your repository path and hosts if they are private. -->

1.
2.
3.

## Expected Behavior

<!-- What you expected to happen -->

## Actual Behavior

<!-- What actually happened -->

## Environment

- **OS**: <!-- e.g., macOS 26.0, Ubuntu 24.04 -->
- **sbx version**: <!-- Run `sbx version` -->
- **Kit version**: <!-- The tag or commit you pinned, or "local clone" -->

## Policy Log

<!-- Run `sbx policy log <sandbox>` and paste the output. It shows which hosts
     were blocked, which is the cause of most kit failures. -->

```
paste output here
```

## Logs/Output

<!-- Never paste an entitlement token, an API key, a pre-signed download URL,
     or a Location header. Any of those leaks a credential. -->

```
paste logs here
```

## Additional Context

<!-- Add any other context about the problem here -->
