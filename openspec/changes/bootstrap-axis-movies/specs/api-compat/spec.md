# api-compat

## ADDED Requirements

### Requirement: Radarr v3 system status
The system SHALL expose `GET /api/v3/system/status` returning a Radarr-compatible
status document so ecosystem tools recognize the application.

#### Scenario: Tool queries status with a valid API key
- **WHEN** a client sends `GET /api/v3/system/status` with a valid `X-Api-Key`
- **THEN** the response is `200` JSON containing `appName` (default `"Radarr"`),
  a Radarr-compatible `version`, and `databaseType: "postgreSQL"`
- **AND** it includes `axisApp: "axis-movies"` identifying the true implementation

#### Scenario: Configurable app identity
- **WHEN** `compat_app_name` is configured to a non-default value
- **THEN** `appName` reflects the configured value

### Requirement: API key authentication
The system SHALL require a valid API key for all `/api/v3` endpoints, accepting it
via the `X-Api-Key` header or an `apikey` query parameter, matching Radarr clients.

#### Scenario: Missing or wrong key is rejected
- **WHEN** a client calls any `/api/v3` endpoint without a valid key
- **THEN** the response is `401 Unauthorized`

#### Scenario: Liveness probe is unauthenticated
- **WHEN** a client calls `GET /ping`
- **THEN** the response is `200` regardless of API key

### Requirement: Read endpoints for ecosystem integration
The system SHALL expose Radarr v3 read endpoints (`/movie`, `/rootfolder`, `/tag`,
`/qualityprofile`, `/indexer`, `/downloadclient`, `/health`) so Prowlarr and
request tools can connect.

#### Scenario: Prowlarr connects Axis as a Radarr application
- **WHEN** Prowlarr adds Axis using its URL and API key
- **THEN** Prowlarr's connection test succeeds against `/api/v3/system/status`
- **AND** indexer sync targets resolve via the `/api/v3/indexer` endpoint
