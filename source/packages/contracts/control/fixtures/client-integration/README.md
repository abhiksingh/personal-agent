# Client Integration Fixture Pack

Canonical fixture payloads for client decode/contract integration are stored here.

- Manifest: `manifest.json`
- API version: `v1`
- Fixture scope:
  - Daemon lifecycle `control_auth` states (`configured`, `missing`)
  - Task action-availability metadata defaults
  - Channel/connector `config_field_descriptors` metadata defaults

Use `tools/scripts/check_client_integration_fixtures.sh` to validate fixture-pack integrity and referenced contract runners.
