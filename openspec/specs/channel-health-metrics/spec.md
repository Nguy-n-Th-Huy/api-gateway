# channel-health-metrics Specification

## Purpose
Gives administrators a per-channel view of how upstream channels are actually behaving under real traffic, so a failing or slow channel is visible in the console instead of being discovered through user complaints.

## Requirements

### Requirement: Channel health metrics derived from request logs

The system SHALL derive per-channel health metrics from recorded request logs over a trailing 24-hour window ending at the time of the request. The system SHALL count a consume log entry as a successful request and an error log entry as a failed request. The system SHALL NOT require any schema change, and SHALL NOT alter routing, retry, auto-disable, billing, or quota behavior as a result of these metrics.

For each channel that has at least one request in the window, the system SHALL report:

- total request count,
- successful request count,
- failed request count,
- success rate, as successful requests divided by total requests,
- average latency, as the sum of recorded request durations divided by total requests.

#### Scenario: Channel with mixed successes and failures

- **WHEN** a channel recorded 8 consume logs and 2 error logs within the last 24 hours
- **THEN** the reported total request count is 10, successful count is 8, failed count is 2, and the success rate is 0.8

#### Scenario: Channel with only failures

- **WHEN** a channel recorded 5 error logs and no consume logs within the last 24 hours
- **THEN** the reported success rate is 0 and the failed count is 5

#### Scenario: Channel with no traffic in the window

- **WHEN** a channel recorded no logs within the last 24 hours
- **THEN** the response contains no metrics entry for that channel

#### Scenario: Requests outside the window are excluded

- **WHEN** a channel's only logs are older than 24 hours
- **THEN** the response contains no metrics entry for that channel

#### Scenario: Log entries not attributable to a channel are excluded

- **WHEN** log entries exist whose channel identifier is absent or non-positive
- **THEN** those entries are excluded from every channel's counts

#### Scenario: Log entry types other than consume and error are excluded

- **WHEN** top-up, management, system, refund, or login log entries exist within the window
- **THEN** those entries are excluded from every channel's counts

### Requirement: Average latency is reported in milliseconds with a stated precision limit

The system SHALL report average latency in milliseconds. Because recorded request durations are stored with one-second granularity, requests completing in under one second are recorded as zero duration; the reported average is therefore a lower bound rather than an exact mean. The system SHALL disclose this limitation to the operator in the console rather than presenting the value as exact.

#### Scenario: Latency converted from seconds to milliseconds

- **WHEN** a channel recorded 4 requests with durations of 1, 2, 3, and 2 seconds
- **THEN** the reported average latency is 2000 milliseconds

#### Scenario: Sub-second requests reported as zero

- **WHEN** a channel's requests all completed in under one second
- **THEN** the reported average latency is 0 milliseconds

#### Scenario: Operator is informed of the precision limit

- **WHEN** an administrator views the average latency column in the channel management table
- **THEN** the console discloses that the value is an approximation derived from one-second-granularity data

### Requirement: Administrator-only access to channel health metrics

The system SHALL expose channel health metrics through an authenticated administrative endpoint guarded by the same permission that governs reading channel data. The system SHALL reject requests from callers lacking that permission.

#### Scenario: Administrator with channel read permission

- **WHEN** an administrator holding the channel read permission requests channel health metrics
- **THEN** the system returns the metrics for all channels with traffic in the window

#### Scenario: Caller without channel read permission

- **WHEN** a caller lacking the channel read permission requests channel health metrics
- **THEN** the system denies the request and returns no metrics

### Requirement: Metrics remain available when logs are stored separately

The system SHALL compute channel health metrics without joining the log store to the channel store, so the feature keeps working when request logs are configured to live in a separate database. The aggregation SHALL be expressed so that it executes on every supported log store.

#### Scenario: Logs configured in a separate database

- **WHEN** request logs are configured to a separate log database
- **THEN** channel health metrics are still computed and returned

### Requirement: Repeated console reads do not repeatedly scan the log store

The system SHALL serve channel health metrics from a short-lived cached aggregation, so that repeated console refreshes within the cache lifetime do not each trigger a full aggregation over the log store.

#### Scenario: Second request within the cache lifetime

- **WHEN** channel health metrics are requested twice within the cache lifetime
- **THEN** the second request is served from the cached aggregation and returns the same values

#### Scenario: Request after the cache lifetime expires

- **WHEN** channel health metrics are requested after the cache lifetime has elapsed
- **THEN** the aggregation is recomputed and reflects logs recorded since the previous computation

### Requirement: Channel management table displays health metrics

The channel management console SHALL display a success rate column and an average latency column for each channel. For a channel with no traffic in the window, the console SHALL display an explicit no-data indicator rather than a zero percentage, so an idle channel is not mistaken for a failing one. All column headings and labels SHALL be available in both English and Vietnamese.

#### Scenario: Channel with traffic

- **WHEN** an administrator views a channel that served requests in the last 24 hours
- **THEN** the success rate and average latency for that channel are displayed

#### Scenario: Channel without traffic

- **WHEN** an administrator views a channel that served no requests in the last 24 hours
- **THEN** the console displays a no-data indicator in both columns, and not `0%`

#### Scenario: Metrics unavailable

- **WHEN** the health metrics cannot be retrieved
- **THEN** the channel list still renders and the two metric columns display the no-data indicator

#### Scenario: Vietnamese interface

- **WHEN** the console language is Vietnamese
- **THEN** both column headings and the no-data indicator are shown in Vietnamese
