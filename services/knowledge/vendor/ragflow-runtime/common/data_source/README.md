# Data Source Connectors

This directory is retained as upstream reference code for future multi-source
knowledge ingestion adapters.

Current boundary:

- The connector implementations are not wired into the active Knowledge runtime.
- The upstream connector HTTP API, service layer, and default tests remain
  removed from this trimmed vendor snapshot.
- Connector-specific third-party dependencies are intentionally not listed in
  the default `pyproject.toml`.

Future integration should first define explicit contracts for:

- Knowledge ingestion ownership and dataset/document mapping.
- File/object access through this repository's File service boundary.
- Authentication and secret handling through this repository's Auth boundary.
- Background sync and indexing jobs through an explicit worker/job contract.
- Optional dependency groups for connector families that are actually enabled.
