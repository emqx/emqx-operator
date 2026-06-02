# HOCON Compliance Spec

This document describes the value model for the compliance suite. The actual machine-readable cases live as `.hocon` files in `compliance/` directory.

These test cases were generated using https://github.com/emqx/hocon as the reference implementation, which deviates in few minor aspects from the upstream specification.

Upstream specification: https://github.com/lightbend/config/blob/main/HOCON.md

## Expectation Format

Each test case is a HOCON file with header comments. Successful cases use:

```hocon
# @expect ok
# @expect-json {"key":"value"}
```

Error cases use:

```hocon
# @expect error parse_error
```

Files without an `@expect` header are helper files for include tests and are not run directly.

For error cases, only the error type is asserted. Exact message text and line formatting are implementation-defined.

## Deviations

The `deviation/` case directory documents current behavior that differs from upstream Lightbend HOCON. Treat those cases as compatibility requirements for this dialect, not as claims about the upstream specification.
