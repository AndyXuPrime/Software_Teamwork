# Security Policy

This is a public repository for a collaborative coursework project: an electric power industry knowledge management system.

## Supported Branches

Security issues are reviewed on a best-effort basis during the course or project period.

| Branch | Support status |
| --- | --- |
| `develop` | Actively reviewed on a best-effort basis |
| `main` | Reviewed for released coursework snapshots, if applicable |
| Personal forks or feature branches | Not officially supported |
| Old or abandoned branches | Not supported |

## Reporting a Vulnerability

Please do not open a public GitHub issue for security vulnerabilities.

Use GitHub private vulnerability reporting for this repository when available. If it is not available, contact the project maintainers directly through the course or team communication channel.

When reporting a vulnerability, please include:

- The affected service, page, workflow, or configuration
- Steps to reproduce the issue
- The potential impact
- Relevant logs, screenshots, or request examples, if safe to share
- A suggested fix, if available

## Scope

Relevant security reports may include issues in:

- Authentication or authorization behavior
- File upload, parsing, storage, or download handling
- API access control across the gateway and backend services
- Secrets, tokens, credentials, or unsafe configuration committed to the repository
- Docker, deployment, CI, or dependency security risks

This project is not a production service and does not operate a bug bounty program. Reports are reviewed by the student maintainers on a best-effort basis.

## Public Disclosure

Please avoid publicly disclosing vulnerability details until the maintainers have had a reasonable opportunity to review and address the issue.
