## [2.0.0] - 2026-09-09

### 🚀 Features

- *(shares)* [**breaking**] Limit shares by number of opens
- *(http)* Allow disabling OpenAPI/Swagger UI endpoints

### 🧪 Testing

- *(back)* Remove t.Parallel from server tests to avoid race
## [1.1.0] - 2026-09-08

### 🚀 Features

- *(health)* Add liveness and readiness probes
- *(shares)* Expose share limits to the client

### 🐛 Bug Fixes

- *(docker)* Create /data owned by app so named volumes are writable

### 📚 Documentation

- *(readme)* Add UI demo GIF and screenshots
## [1.0.0] - 2026-09-08

### 🚀 Features

- Initial release of sharelock

### ⚙️ Miscellaneous Tasks

- Pin third-party actions, add frontend type-check, trim docker context
- Make release.sh executable
