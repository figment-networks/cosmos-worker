# Change Log

## [0.2.1] - 2021-04-16

### Added

### Changed
### Fixed
- removed wrongly attributed rewards to unbondedTokensPoolAddr in `undelegate` transactions

## [0.2.0] - 2021-04-14

### Added
- Add error msg from rawlog to Sub.Error
### Changed
- deprecated lcd, the following config variables are no longer needed: `tendermint_lcd_addr`, `requests_per_second_lcd`, `datahub_key`
- getRewards now returns validator info, compatible with latest version of manager
### Fixed

## [0.1.6] - 2021-03-18

### Added
- Context bound to live calls

### Changed
- Bumped cosmos-sdk library to 0.42.1
- Unify metrics among workers
### Fixed

## [0.1.5] - 2021-03-08

### Added
- Configurable timeouts for grpc connections
- Vesting message type

### Changed
- Bumped cosmos-sdk library to 0.42.0
### Fixed


## [0.1.4] - 2021-03-08

### Added
### Changed
### Fixed
- Wrong source validator in begin_redelegate [https://github.com/figment-networks/cosmos-worker/pull/19](PR #19)


## [0.1.3] - 2021-03-04

### Added
### Changed
- go.mod file
### Fixed

## [0.1.2] - 2021-03-04

### Added
- Adds method to fetch account balance for account
### Changed
### Fixed

## [0.1.1] - 2021-02-16

### Added

### Changed
### Fixed
- Added missing ibc transfer type transaction: "transfer"
- Added missing ibc channel type transactions: "channel_open_confirm", "channel_open_ack", "channel_open_try", "channel_close_init", "channel_close_confirm", "recv_packet", "timeout", "channel_acknowledgement"

## [0.1.0] - 2021-02-12

### Added
- Initial release

### Changed
### Fixed
