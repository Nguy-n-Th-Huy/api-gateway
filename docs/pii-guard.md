# PII Guard

The PII guard masks personally identifiable information in the conversation this
gateway forwards to an upstream AI provider, and restores the real values in the
response. The provider only ever sees the stand-ins; the caller still receives
real data.

```
client ──→ [detect] ──→ [replace] ──→ provider sees stand-ins ──→ [restore] ──→ client
             local         local                                        local
```

Everything runs in-process. No text is sent anywhere for detection, and the
real-to-stand-in mapping stays in memory for the lifetime of one request.

The design follows two references:

- [daslabhq/pii-proxy](https://github.com/daslabhq/pii-proxy) — deterministic
  bijective replacement plus a round-trip restore, so a request never carries
  real PII upstream while the caller still sees real values.
- [microsoft/presidio](https://github.com/microsoft/presidio) — layered
  detection with validating checks, so a candidate span is only accepted when it
  also satisfies the format of the entity it claims to be (a payment card must
  pass the Luhn checksum, an address must be well formed, and so on).

## Turning it on

The guard is **off by default**. When it is off, the outbound body is forwarded
byte for byte, exactly as it was before the feature existed.

Configuration lives in the shared option system under the `piiguard.` prefix, so
it is editable from the admin settings UI (Security → PII Guard) and through the
options API:

| Option | Type | Default | Meaning |
| --- | --- | --- | --- |
| `piiguard.enabled` | bool | `false` | Master switch. |
| `piiguard.mask_request` | bool | `true` | Mask message text, prompts and instructions on the way out. |
| `piiguard.unmask_response` | bool | `true` | Restore the real values in the response. Only valid in `pseudonym` mode. |
| `piiguard.mode` | string | `pseudonym` | `pseudonym` = a deterministic stand-in of the same shape; `redact` = a typed placeholder such as `[EMAIL_1]`. |
| `piiguard.placeholder_style` | string | `typed` | Placeholder shape in `redact` mode: `typed` (`[EMAIL_1]`) or `template`. |
| `piiguard.token_template` | string | `«{{type}}_{{index}}»` | Placeholder pattern used by `placeholder_style=template`. |
| `piiguard.secret` | string | empty | Seeds the deterministic generator. Empty means a per-process random seed. |
| `piiguard.max_body_bytes` | int | `1048576` | Bodies larger than this are not rewritten. |
| `piiguard.enabled_entity_types` | string[] | empty | Restrict detection to these entity types. |
| `piiguard.disabled_entity_types` | string[] | empty | Remove these entity types. Wins over the enabled list. |
| `piiguard.custom_keywords` | string[] | empty | Literals that are always masked as `CUSTOM`. |
| `piiguard.min_keyword_length` | int | `3` | Shortest custom keyword that is masked. |
| `piiguard.fakes` | object | empty | Override the generated stand-in: entity type → real value → replacement. |
| `piiguard.require_mask_reject` | bool | `false` | Fail the request instead of forwarding a body that could not be masked. |

Two combinations are rejected when saved: an enabled guard with neither request
masking nor response unmasking (it would only cost CPU), and `redact` mode with
`unmask_response` (redaction placeholders are not meant to be reversed
downstream).

`piiguard.secret` is a write-only option: the options API withholds every key
ending in `Secret` or `secret`, so the settings form shows an empty field whose
value is only a seed for the generated stand-ins and nothing else.

## Modes

**`pseudonym`** (default) replaces a value with a deterministic stand-in of the
same shape:

```
Contact alex@example.com about invoice 4111 1111 1111 1111
Contact pii-12d7d456ebe1@example.com about invoice 0000339947507399
```

The stand-in is derived from the seed plus the real value, so the same real
value always produces the same stand-in inside one request. That is what lets
the model keep following a referent across messages — it sees one consistent
entity, not a new one every turn. Generated values deliberately stay inside
reserved or documentation ranges (RFC 2606 domains, RFC 5737 documentation
addresses, the 555-01XX phone block, the `0000` card prefix, the
`2001:db8::/32` IPv6 block) so a stand-in can never collide with a real,
reachable destination.

**`redact`** replaces a value with a typed placeholder:

```
Contact [EMAIL_1] about invoice [CREDIT_CARD_1]
```

Each distinct real value gets its own placeholder, so two entities stay
distinguishable. Redaction is a stronger privacy posture, but it is also blunter:
the request-level mapping cannot be reversed through the response the way a
pseudonym can, and `unmask_response` is therefore forced off.

## What is detected

| Entity | Detection | Notes |
| --- | --- | --- |
| `EMAIL` | Pattern | |
| `PHONE` | Pattern | International form always; the local `0…` form needs a context word such as "phone", "điện thoại", "sđt". |
| `CREDIT_CARD` | Pattern + Luhn | A 13–19 digit run is only a card when its checksum is valid. |
| `IP_ADDRESS` | Pattern | IPv4 and IPv6. |
| `UUID` | Pattern | |
| `IBAN` | Pattern | Needs a banking context word, because the shape alone matches other identifiers. |
| `PASSPORT` | Pattern + context | |
| `DRIVER_LICENSE` | Pattern + context | |
| `VN_ID` | Pattern + context | 9 or 12 digits behind "CCCD", "CMND", "căn cước", "citizen id". |
| `DATE_OF_BIRTH` | Pattern + context | |
| `CUSTOM` | Literal | From `piiguard.custom_keywords`. |
| `URL` | Pattern | Off by default. |
| `US_SSN` | Pattern | Off by default. |
| `VN_TAX_CODE` | Pattern + context | Off by default. |
| `PERSON` | Lexicon + salutation | Off by default; see the limitation below. |
| `LOCATION`, `ORGANIZATION` | Context keyword | Off by default. |

Overlapping candidates are resolved longest-match-first, with the more specific
entity winning a tie. That is the rule that keeps an email address from being
reported as a URL plus its local part.

When the guard is on, every replacement is logged by count and entity type only,
and the consume log carries the same counts plus the JSON field paths that were
rewritten under `other.admin_info.pii_masking`. The real-to-stand-in mapping is
never written to a log, a metric or the database: it is the sensitive artifact,
and it lives only as long as the request.

## Limitations

Detection is local and pattern-based. It is not a trained NER model, so:

- **Person names are not reliably detectable.** `PERSON` uses a salutation
  heuristic plus a small Vietnamese given-name lexicon, and it is off by
  default. Turn it on only after testing it against your own traffic.
- **Novel identifiers can slip through.** A regulated number with no general
  pattern belongs in `piiguard.custom_keywords`.
- **Values below `piiguard.max_body_bytes` are rewritten through a JSON walk.**
  A body above the limit is forwarded untouched, or rejected when
  `piiguard.require_mask_reject` is on. That trade-off exists so a large
  multimodal payload is never expanded into a generic map in memory.
- **Only the converted relay path is masked.** When a channel or the global
  setting enables request pass-through, the client body is streamed upstream
  unchanged and the guard does not rewrite it.
- **Streaming responses are restored at the byte level.** A stand-in split
  across two SSE chunks is held back until it is complete, so the restore is
  exact, but a stand-in the model invented by itself (rather than reusing one it
  was given) is not restored.

## Implementation

| Path | Responsibility |
| --- | --- |
| `pkg/piiguard/detectors.go` | The detector layer: patterns, context gates, priorities. |
| `pkg/piiguard/generate.go` | Deterministic format-preserving stand-in generators. |
| `pkg/piiguard/engine.go` | Per-request engine: detection, overlap resolution, the bijective mapping. |
| `pkg/piiguard/body.go` | JSON body walk over the text fields, and the field-path recording. |
| `pkg/piiguard/unmask.go` | Streaming restore with a hold-back window for split stand-ins. |
| `setting/piiguard_setting/` | Option registration, load/save and validation. |
| `service/pii_guard.go` | The request-scoped filter the relay uses. |

The relay handlers mask the outbound body last, after field removal and
parameter override, so what goes upstream is exactly what was masked. The
response body is wrapped once, where the upstream response is received, so every
downstream reader — streaming and non-streaming formats alike — sees real data
again.
