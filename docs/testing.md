# Testing Guidelines

- Prefer testing exported behavior over testing private helpers directly.
- Do not mock loggers or assert on logging side effects in tests unless the logging itself is the exported behavior.
- Configure mocks/stubs only to provide the scenario. Prefer to not assert argument values, call order, or interactions.
- Add integration coverage when behavior crosses HTTP, database, or migration boundaries.
- Assert observable results: returned value or error, and relevant domain state changes.

## Go Testing Guidelines

- Use `testify` for assertions and suites.
- Use `require` for prerequisite checks and `assert` for follow-up state checks.
- For error assertions, prefer `require.Error`, `require.NoError`, `require.ErrorIs`, `require.ErrorContains`, and
  `require.ErrorAs`.
- Name tests `Test<StructName>_<MethodName>_<AdditionalInfo>` for struct tests.
- Name tests `Test<FunctionName>_<AdditionalInfo>` for function tests.
- Keep fixtures under `testdata/` where applicable.
- Prefer table tests, but split them into separate tests when tables become harder to read than the behavior they cover.
- Do not use `t.Parallel()` without approval.
- Use `t.Context()` instead of `context.Background()` in tests.
