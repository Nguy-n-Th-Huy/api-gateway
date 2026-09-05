# Feature Gap Analysis: api-gateway (new-api fork) vs xpiki.com

Ngày: 2026-09-05 | Phạm vi: đối chiếu tính năng, không sửa code sản phẩm.

## 0. Đã đọc AGENTS.md + web/AGENTS.md

Ràng buộc bắt buộc rút ra (áp dụng cho mọi đề xuất bên dưới):

- DB: mọi thay đổi phải chạy được trên SQLite + MySQL >=5.7.8 + PostgreSQL >=9.6; migration dùng `ALTER TABLE ADD COLUMN` (không `ALTER COLUMN` trên SQLite); dùng `lockForUpdate(tx)`, `commonGroupCol/commonKeyCol`, `common.UsingMainDatabase/UsingLogDatabase`.
- JSON: bắt buộc `common.Marshal/Unmarshal/...` trong `common/json.go`, cấm `encoding/json` trực tiếp trong business code.
- Billing safety: mọi quantity user-controlled thành multiplier phải bound trước khi vào quota calc; quy đổi quota chỉ qua `common/quota_math.go` (`QuotaFromFloat/QuotaRound/QuotaFromDecimal` + biến thể `*Checked`); không tạo helper convert cục bộ; saturation phải audit qua `attachQuotaSaturation`.
- `relaykit/` phải build độc lập: `cd relaykit && GOWORK=off go build ./...`.
- i18n bắt buộc en/vi cả BE (`i18n/`, go-i18n) lẫn FE (`web/src/i18n/locales/{lang}.json`, `useTranslation()`), không hardcode text.
- Không đụng branding "new-api" / "QuantumNous" (bảo vệ tuyệt đối).
- FE: tuân `web/AGENTS.md` — feature module tại `src/features/<feature>/`, React Query + Zustand, RHF+Zod, TanStack Router, test bắt buộc khi đổi hành vi UI.

## 1. Đã có tương đương (tránh đề xuất trùng)

| Tính năng xpiki | Trạng thái repo | Bằng chứng |
|---|---|---|
| Devices & Sessions | CÓ, khá mạnh (session control-plane cho JWT, revoke, refresh-reuse detection) | `model/user_session.go`; `web/src/features/profile/components/login-sessions-card.tsx`, `login-session-item.tsx`, `login-session-dialogs.tsx` |
| Audit Logs | CÓ | `controller/audit.go` (auditContentTemplates cho user/channel/redemption/subscription...), `middleware/audit.go` |
| Models Hub (filter, price, cards/table, model detail, uptime/perf) | CÓ, đầy đủ | `web/src/features/pricing/` (`model-card-grid.tsx`, `pricing-table.tsx`, `pricing-toolbar.tsx`, `model-details-*.tsx`, `model-details-uptime-sparkline.tsx`, `model-perf-badge.tsx`) |
| Groups + multiplier/ratio | CÓ | `setting/ratio_setting/`, `web/src/features/system-settings/models/group-ratio-form.tsx`, `group-ratio-visual-editor.tsx`, `group-special-usable-editor.tsx`; auto-group `setting/auto_group.go` |
| Billing & Recharge (SePay) | CÓ, đã thay 5 gateway (commit `4e56aeb5e`) | `controller/sepay.go`, `model/sepay.go`, `web/src/features/wallet/components/sepay-payment-panel.tsx`, `recharge-form-card.tsx`, `subscription-plans-card.tsx` |
| Redeem Codes | CÓ | `controller/redemption.go`, `model/redemption.go`, `web/src/features/redemption-codes/` |
| Usage/Request logs + drawing logs | CÓ | `controller/log.go`, `web/src/features/usage-logs/` (`drawing-logs-columns.tsx` cho Midjourney/drawing) |
| API Keys (token) quản lý | CÓ | `controller/token.go`, `model/token.go`, `web/src/features/keys/` |
| Public leaderboard mô hình (gần giống Token Race nhưng khác đối tượng) | CÓ MỘT PHẦN | `service/rankings.go`, `web/src/features/rankings/components/model-leaderboard.tsx` — xếp hạng **model/vendor theo token toàn hệ thống**, KHÔNG xếp hạng người dùng, không có Achievements |
| Channel pool cơ bản (nhiều key/channel, priority/weight, auto-disable, upstream sync) | CÓ, nhưng khác kiến trúc "Provider Account Pool" của xpiki | `model/channel.go` (Weight/Priority/AutoDisabled), `controller/channel-test.go` (nút Test), audit action `channel.multi_key_manage`, `channel.upstream_apply_all` |
| White-label cơ bản (site name/logo/footer/about/custom home page) | CÓ, đơn-domain | `web/src/features/system-settings/site/index.tsx` (SystemName/Logo/Footer/About/HomePageContent), `web/src/features/home/` (redesign VN, commit `fc4046fd8`) |
| Referral/affiliate | CÓ (không có trong khảo sát xpiki) | `web/src/features/wallet/components/affiliate-rewards-card.tsx` |
| i18n en/vi song ngữ + hiển thị giá theo tiền tệ khách | CÓ, mới làm (commit `2b72a1b98`) | `web/src/i18n/locales/{en,vi}.json`, hooks liên quan currency trong `web/src/features/pricing` |

## 2. Thiếu / yếu hơn xpiki

| # | Tính năng xpiki | Kết luận | Bằng chứng thiếu |
|---|---|---|---|
| G1 | Playground Chat\|Image\|Video trong 1 khu vực, log đồng bộ Usage | Playground hiện CHỈ có Chat | `web/src/features/playground/` chỉ có `use-chat-handler.ts`, `use-playground-conversation.ts`, `use-stream-request.ts` — không có hook/component image hay video; grep "image generation\|video generation\|Midjourney\|Sora\|Veo\|Runway" trong `web/src/features` không match `features/playground` hay `features/chat` (chỉ match settings/logs, tức backend có hỗ trợ generate ảnh nhưng UI người dùng cuối không có tab Image/Video) |
| G2 | Provider Account Pool: Meter/Dispatch, usage req + success rate + latency trung bình theo từng tài khoản upstream, Overage toggle, Pool Health tab, Custom Providers tab, bulk import/delete tài khoản | KHÔNG CÓ mô hình pool riêng | grep `ProviderAccount\|provider_account\|AccountPool` toàn repo: 0 kết quả. Channel hiện tại (`model/channel.go`) là 1 channel = 1 hoặc nhiều key dạng danh sách phẳng, có Priority/Weight/AutoDisabled nhưng KHÔNG có success-rate/latency trung bình theo account, không có Overage toggle, không có "Pool Health" tab riêng |
| G3 | Sell Keys (bán key kèm credential thật, không cần login, sticky binding) + Sell pool / Proxy pool / Sell proxy tabs | KHÔNG CÓ | grep `SellKey\|sell_key\|sell-key\|SellCredential`: 0 kết quả trong toàn repo (Go + web) |
| G4 | Token Race: leaderboard công khai xếp hạng NGƯỜI DÙNG theo usage trong 1 window + tab Achievements (gamification) | KHÔNG CÓ (chỉ có leaderboard MODEL, xem mục 1) | grep `TokenRace\|token_race\|achievement` (không phân biệt hoa thường) toàn repo: 0 kết quả. `service/rankings.go` chỉ tổng hợp theo `ModelName`/vendor, không có `UserID` trong `RankedModel`/`RankedVendor` |
| G5 | Proxies (danh sách proxy riêng cho outbound) | KHÔNG THẤY module riêng | grep `SellKey...Proxy pool` không match; không tìm thấy `model/proxy.go` hay `controller/proxy.go` tương đương "Proxies" menu của xpiki (ngoài phạm vi request nên chỉ note nhẹ, chưa xác minh sâu — cần xác nhận thêm) |
| G6 | White-label ĐA DOMAIN cho nhiều brand trên 1 instance (mỗi domain: logo/tên/site riêng) | Repo chỉ hỗ trợ 1 bộ site setting (đơn-tenant) | `defaultSiteSettings` trong `web/src/features/system-settings/site/index.tsx` là 1 object phẳng (SystemName/Logo/Footer duy nhất), không có khái niệm site-per-domain/tenant; không tìm thấy `CustomDomain`/multi-tenant trong toàn repo |
| G7 | "Mở thẳng Playground" từ Models Hub / bảng giá | Không thấy nút liên kết | grep "playground" trong `pricing-columns.tsx`, `model-card.tsx`: 0 kết quả (cần xác minh thêm ở các file card/detail khác nếu muốn chắc chắn 100%, nhưng không thấy ở 2 điểm chính hiển thị model) |
| G8 | Provider Accounts: bulk import/delete tài khoản upstream hàng loạt | Repo có "Custom Providers/Ollama models dialog" (`ollama-models-dialog.tsx`) nhưng ở cấp channel/model, không phải cấp tài khoản-pool | Không tìm thấy `bulk import` account cho channel trong `controller/channel.go` (grep "bulk" không match) — chỉ có bulk theo tag (`channel.tag_batch_set`, `channel.delete_batch`) |

## 3. Đề xuất nâng cấp (theo từng gap)

### P0 — Giá trị cao / chi phí hợp lý cho thị trường VN

**P0-1. Playground: thêm tab Image + Video (mở rộng từ Chat hiện có)**
- Mô tả: thêm 2 tab trong `web/src/features/playground/` để user tự test sinh ảnh/video ngay trong Playground, log tự động vào Usage (đã có `usage-logs` + `drawing-logs-columns.tsx`).
- Module ảnh hưởng: `web/src/features/playground/` (mới: `use-image-handler.ts`, `use-video-handler.ts`, components tab), `web/src/features/playground/types.ts`; backend đã có `controller/image.go`, `controller/task.go`, `controller/midjourney.go` — chủ yếu là request FE mới gọi API sẵn có, ít việc backend.
- Độ khó: **M** (chủ yếu FE; nếu cần thêm endpoint video status polling thì +S backend).
- Blast radius: thấp — module mới trong `features/playground`, không đổi contract API hiện có nếu tái dùng endpoint image/task đã có.
- Ràng buộc AGENTS.md: dùng `useTranslation()` + thêm key vào `web/src/i18n/locales/{en,vi}.json`; giữ nguyên các invariant billing khi hiển thị estimate cost (image `n`, video `duration`) — PHẢI dùng lại `dto.MaxImageN`, `relaycommon.MaxTaskDurationSeconds` nếu FE hiển thị estimate, KHÔNG tạo giới hạn mới; test mới đặt trong `__tests__/` theo module.
- Lý do ưu tiên P0: xpiki lấy Playground đa phương tiện làm điểm bán hàng chính; thị trường VN thường dùng để demo nhanh trước khi mua — chi phí thấp vì tận dụng backend có sẵn.

**P0-2. Nút "Mở Playground" trực tiếp từ Models Hub/Pricing**
- Mô tả: thêm CTA trong `model-card.tsx` / `pricing-columns.tsx` để điều hướng sang Playground với model được pre-select.
- Module: `web/src/features/pricing/components/model-card.tsx`, `pricing-columns.tsx`, `web/src/features/playground/` (nhận query param model).
- Độ khó: **S**.
- Blast radius: rất thấp, chỉ FE routing (`TanStack Router` — `useNavigate`/`Link`).
- Ràng buộc: dùng i18n cho label nút; không dùng `window.location`.
- Lý do P0: chi phí cực thấp, tăng conversion trực tiếp (đúng insight từ commit gần đây tối ưu trang chủ/giá cho VN).

### P1 — Giá trị trung bình-cao, chi phí trung bình

**P1-1. Provider Account observability nâng cao (success rate + latency trung bình theo channel/key, không cần đổi hẳn kiến trúc thành "pool")**
- Mô tả: KHÔNG đề xuất viết lại channel thành "Provider Account Pool" (chi phí L, rủi ro breaking rất lớn với billing/relay). Thay vào đó bổ sung 2 chỉ số (success rate, avg latency) vào channel hiện có, hiển thị trong bảng Channels + tab "Pool Health"-style filter.
- Module: `model/channel.go` (thêm trường tổng hợp hoặc bảng phụ `channel_health_stat`), `service/` (job tính rolling success-rate/latency từ log hiện có `model/log.go`), `web/src/features/channels/` (cột mới trong bảng, dialog test hiện có `channel-test-dialog.tsx`).
- Độ khó: **M-L** tuỳ có tạo bảng mới hay tính on-the-fly từ log.
- Blast radius: trung bình — đụng `model/channel.go` (schema) và luồng relay khi cần ghi nhận latency; PHẢI qua đủ 3 DB test theo AGENTS.md nếu thêm cột/bảng, dùng `ALTER TABLE ADD COLUMN` (không `ALTER COLUMN`), test migration idempotent 2 lần.
- Ràng buộc: nếu latency ảnh hưởng quyết định billing/quota thì tuyệt đối không, đây chỉ là số liệu quan sát (observability), không đụng `common/quota_math.go`.
- Lý do P1 (không P0): giá trị thật (giúp admin/VN vận hành pool multi-key tốt hơn) nhưng chi phí không nhỏ vì đụng schema + cần xác minh 3 DB.

**P1-2. Token Race dạng nhẹ: leaderboard người dùng (opt-in, ẩn danh hoá) theo tuần**
- Mô tả: mở rộng `service/rankings.go` thêm chế độ xếp hạng theo `UserID` (đã có sẵn khung period/cache), hiển thị ở trang public mới (không phải Achievements đầy đủ ở P1 này).
- Module: `service/rankings.go` (thêm `RankedUser` struct, cần opt-in flag trong `model/user.go` vì lộ usage cá nhân là vấn đề riêng tư), `web/src/features/rankings/`.
- Độ khó: **M**.
- Blast radius: trung bình — cần thêm cột opt-in cho user (schema change, 3-DB test), cần cân nhắc riêng tư (không mặc định bật).
- Ràng buộc: opt-in bắt buộc để tránh vấn đề riêng tư/PII; i18n cho toàn bộ text; không đổi cấu trúc `RankingsResponse` hiện có cho model/vendor (giữ backward-compat).
- Lý do P1: gamification giúp giữ chân người dùng trẻ ở VN (thị trường game hoá mạnh) nhưng cần thêm quyết định sản phẩm về privacy trước khi làm — không thể ưu tiên P0 vì cần user xác nhận hướng đi.

**P1-3. White-label: cho phép nhiều tên miền cùng trỏ instance với site setting khác nhau (multi-brand nhẹ)**
- Mô tả: mở rộng `system_setting`/`console_setting` để chọn site settings theo `Host` header, không cần multi-tenant DB đầy đủ.
- Module: `setting/system_setting/` hoặc `setting/console_setting/`, `middleware/` (resolve theo Host), `web/src/features/system-settings/site/`.
- Độ khó: **L**.
- Blast radý: trung bình-cao — đụng middleware toàn cục, cần cache theo host, ảnh hưởng mọi request.
- Ràng buộc: đây là thay đổi kiến trúc, cần brainstorm/plan riêng trước khi code (theo primary-workflow); không đụng branding new-api/QuantumNous dù cho phép brand khác qua site setting.
- Lý do P1 chứ không P0: giá trị cao cho mô hình reseller ở VN nhưng độ khó/rủi ro lớn, cần xác nhận nhu cầu kinh doanh thật (xem câu hỏi mục 5).

### P2 — Giá trị thấp hơn cho thị trường VN hiện tại / chi phí cao hoặc rủi ro kiến trúc

**P2-1. Sell Keys / Sell pool / Proxy pool / Sell proxy (bán credential không cần login)**
- Mô tả: module hoàn toàn mới — bán trực tiếp key/credential kèm proxy, sticky binding IP.
- Module: mới hoàn toàn — `model/`, `controller/`, `service/`, `router/`, `web/src/features/sell-keys/` (không tồn tại).
- Độ khó: **L** (module mới, liên quan billing, bảo mật credential, chống gian lận).
- Blast radius: cao nếu tích hợp với billing/quota hiện có; thấp nếu làm độc lập nhưng vẫn cần thiết kế bảo mật riêng (rò rỉ credential thật cho người không đăng nhập là rủi ro lớn).
- Ràng buộc: PHẢI qua brainstorm/threat-model riêng trước (theo `review-audit-self-decision.md` — threat model bắt buộc trước khi code); billing an toàn theo `common/quota_math.go` nếu bán theo quota.
- Lý do P2: mô hình kinh doanh khác biệt lớn so với hướng hiện tại của repo (SePay + subscription cho end-user đăng nhập), rủi ro lạm dụng/pháp lý cao, cần quyết định kinh doanh trước khi đầu tư kỹ thuật.

**P2-2. Achievements (gamification badges) cho Token Race**
- Mô tả: mở rộng thêm từ P1-2, thêm hệ thống badge/thành tựu.
- Module: `model/` (bảng achievements mới), `web/src/features/rankings/`.
- Độ khó: **M**.
- Blast radius: thấp nếu độc lập với billing.
- Lý do P2: phụ thuộc P1-2 được chấp nhận trước; giá trị marginal thấp hơn leaderboard cơ bản.

**P2-3. Proxies (module quản lý proxy outbound độc lập)**
- Mô tả: cần xác nhận thêm phạm vi (proxy cho relay ra ngoài hay proxy bán cho khách) trước khi ước lượng — hiện chưa xác minh sâu được trong phạm vi thời gian audit này.
- Độ khó: chưa xác định — cần thêm phân tích.
- Lý do P2/hoãn: thiếu thông tin để đánh giá chi phí/giá trị chính xác (xem câu hỏi mục 5).

## 4. Bảng ưu tiên tóm tắt

| Ưu tiên | Đề xuất | Độ khó | Lý do |
|---|---|---|---|
| P0 | P0-1 Playground Image+Video | M | Tận dụng backend có sẵn, giá trị demo/bán hàng cao |
| P0 | P0-2 CTA Playground từ Pricing | S | Chi phí cực thấp, tăng conversion |
| P1 | P1-1 Channel success-rate/latency | M-L | Giá trị vận hành thật, chi phí schema+3DB verify |
| P1 | P1-2 Leaderboard người dùng (opt-in) | M | Gamification nhẹ, cần quyết định privacy trước |
| P1 | P1-3 White-label multi-domain nhẹ | L | Giá trị reseller, rủi ro middleware toàn cục |
| P2 | P2-1 Sell Keys/Proxy pool | L | Mô hình kinh doanh khác, rủi ro pháp lý/bảo mật |
| P2 | P2-2 Achievements | M | Phụ thuộc P1-2 |
| P2 | P2-3 Proxies module | ? | Chưa đủ dữ liệu đánh giá |

## 5. Câu hỏi chưa giải đáp / giả định cần xác nhận

1. **Playground Image/Video (P0-1)**: xác nhận model image/video nào cần hỗ trợ trước (Midjourney đã có relay; Sora/Veo/Runway chưa thấy channel adapter — cần kiểm tra `relay/channel/` riêng nếu muốn hỗ trợ đủ 75 model như xpiki. Phạm vi audit này KHÔNG đọc hết `relay/channel/*` nên chưa chắc 100% những provider nào đã có sẵn adapter).
2. **P1-2 (leaderboard người dùng)**: có chấp nhận rủi ro riêng tư (hiển thị usage cá nhân dù ẩn danh hoá) cho thị trường VN không? Cần quyết định opt-in mặc định tắt/bật.
3. **P1-3 (multi-domain white-label)**: có nhu cầu kinh doanh reseller/multi-brand thật sự chưa, hay chỉ cần tùy biến 1 brand VN là đủ? Đây là thay đổi kiến trúc lớn, không nên làm nếu chưa có nhu cầu cụ thể.
4. **P2-1 (Sell Keys)**: mô hình bán credential không cần đăng nhập có phù hợp định hướng pháp lý/compliance hiện tại (SePay, KYC) không? Cần xác nhận trước khi đầu tư.
5. **G5/P2-3 (Proxies)**: chưa xác minh sâu liệu repo có cơ chế proxy outbound ẩn dưới tên khác (vd. trong `relay/channel/` hoặc `setting/`) — cần audit riêng nếu ưu tiên mục này.
6. **G7**: chỉ kiểm tra 2 file hiển thị model chính (`pricing-columns.tsx`, `model-card.tsx`); chưa quét toàn bộ `model-details-*.tsx` — có thể nút Playground đã tồn tại ở đâu đó trong model detail mà audit chưa phát hiện.
7. Toàn bộ ước lượng "độ khó" là ước lượng định tính từ đọc code, chưa qua brainstorm/plan chi tiết — theo `primary-workflow.md`, các mục M/L nên qua bước brainstorm riêng trước khi implement.

---
Claude-Session: https://claude.ai/code/session_01FN42sbAet7ijV4G22evSrW
