# Task Orchestration Safety

When launching external AI runtimes:

```text
preserve the original prompt
show task status clearly
stream stdout and stderr separately
make cancellation explicit
do not hide failures behind optimistic UI
```

Rendered output is a convenience layer. Raw terminal output must remain inspectable.
