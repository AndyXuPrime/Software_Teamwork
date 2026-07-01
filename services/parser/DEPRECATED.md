# DEPRECATED

This service is **retired** as of the Knowledge vendor runtime replacement (Phase 5).

Document parsing, OCR, and chunking now run in the vendored RAGFlow deepdoc pipeline
(`services/knowledge/vendor/ragflow-runtime/`) behind the Knowledge contract adapter.

The `legacy` compose profile may still build this image for reference; default stacks
do not start it.

Do not add new features here. Remove this directory in a follow-up once CI labels and
docs are fully migrated.
