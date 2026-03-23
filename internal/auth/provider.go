package auth

// This file is intentionally kept minimal.
// All credential types, constants, errors and driver implementations
// live in driver.go. This file existed for EnvSecretProvider which has
// been removed — credentials are now loaded via NewDriverFromEnv with
// file-based secret mount support for zero-downtime rotation.
