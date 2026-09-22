# Export contract fixtures

Synthetic Slack history and expected output pin scli `854e6a0`'s export types
and conversion rules. Time is fixed; failed downloads retain metadata and an
empty `local_path`. The intentional correction is broadcast-reply deduplication.
The late reply precedes an earlier unrelated message because output is grouped
by selected parent, matching scli. No real workspace data or credentials appear.
