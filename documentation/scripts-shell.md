## POSIX
I'm trying to make the scripts POSIX-compliant so they are portable between Linux and macOS.
As a result, some of the shell script code is not the standard code you would see
in a bash script. This section documents some differences.

### Reading user input

To ensure interactivity even if stdin is not a TTY (which can happen under runners),
the code reads from /dev/tty.

### POSIX shell parameter expansion

- `${VAR?}` means: if VAR is unset, print an error message to stderr and abort the command (and usually the script).
- `${VAR:?}` means: if VAR is unset or empty, print an error and abort.

Variants:
- `${VAR?message}` → error if unset; show custom message.
- `${VAR:?message}` → error if unset or empty; show custom message.
- `${VAR:-default}` → use default if unset or empty (no error).
- `${VAR-default}` → use default if unset (empty is accepted).

Example:
- `foo=""; echo "${foo:?must not be empty}"` → prints “parameter null or not set: must not be empty” and exits with non-zero status.