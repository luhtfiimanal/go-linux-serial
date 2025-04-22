# Changelog

All notable changes to this project will be documented in this file.

## [v1.1.0] - 2025-04-22
### Changed
- Added robust reconnection logic to `SerialReader` via `ReadLinesWithReconnect`, which now retries on error, logs attempts, sleeps between retries, and supports a maximum retry count.
- EINTR (interrupted system call) errors are now handled gracefully by retrying the operation instead of aborting the read loop.
- Introduced a new `Reopen` method to encapsulate safe closing and reopening of the serial port, improving encapsulation and resource management.

## [v1.0.0] - 2025-04-17
### Added
- Initial public release
- Linux-only, killable, low-latency serial port reader
- `SerialReader` with `ReadLine`, `ReadLinesLoop`, `WriteLine`, and `Close`
- PTY-based unit tests for reliability
- MIT License
- GoDoc polish and usage examples
