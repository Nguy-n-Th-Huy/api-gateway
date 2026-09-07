package model

// TelegramCreditedEventHook, when set, is invoked once a SePay top-up credit
// has committed, so the Telegram bot integration (package service) can emit a
// best-effort "credited" event without this package importing service, which
// already imports model and would create an import cycle. Wiring happens in
// main.go, mirroring how i18n.SetUserLangLoader breaks the same kind of
// dependency direction elsewhere in this codebase.
//
// The hook MUST NOT be allowed to delay, fail, or roll back the credit it is
// called after (specs/telegram/bot-events/spec.md, "Delivery failure never
// damages the originating operation"): callers invoke it only after their
// transaction has committed, and every implementation must return without
// blocking on network I/O.
var TelegramCreditedEventHook func(userId int, tradeNo string, creditedAmount int64)
