# Kiểm toán bản địa hóa tiếng Việt — new-api

Ngày: 2026-09-05 · Chế độ: CHỈ ĐỌC (không sửa file nguồn)

Phạm vi kiểm toán:

| Tệp | Số mục | Mức bao phủ |
|---|---|---|
| `web/src/i18n/locales/vi.json` (khóa `translation`) | 5.583 | Quét máy 100% cho placeholder, chuỗi rỗng, chuỗi trùng khóa tiếng Anh, chuỗi không dấu, dấu ngoặc lệch, thương hiệu, và 15 cụm thuật ngữ miền. Đọc thủ công ~450 mục do các bộ lọc đó trả về. Các mục còn lại (đã dịch, có dấu, không trúng bộ lọc) chưa được đọc từng dòng. |
| `i18n/locales/vi.yaml` (backend go-i18n) | 238 | Đọc 100% + đối chiếu placeholder với `en.yaml` |

---

## 1. Tóm tắt

| Lớp lỗi | Số lượng | Mức độ |
|---|---|---|
| Hỏng placeholder / chuỗi bị cắt cụt | 3 | Nghiêm trọng — vỡ render |
| Vi phạm thương hiệu (dịch tên "New API") | 6 | Nghiêm trọng — vi phạm chính sách dự án |
| Dịch sai nghĩa / dịch máy nghĩa đen | 20 | Cao |
| Còn nguyên tiếng Anh (chuỗi hoặc từ) | 41 | Cao |
| Thuật ngữ không nhất quán | 92 | Trung bình |
| Ngữ pháp / văn phong / dấu tiếng Việt | 17 | Thấp–Trung bình |
| **Tổng** | **179** | |

**Đánh giá chung.** `vi.json` là bản dịch máy có hậu biên tập một phần: phần lớn câu dài đọc tự nhiên và đủ dấu, nhưng lớp nhãn ngắn (nút, tiêu đề cột, tab) bị bỏ sót nặng và bộ thuật ngữ miền chưa được chuẩn hóa — riêng từ *model* có 4 cách dịch, *token* có 2, *quota* có 3, *upstream* có 3. Không có mục nào bị mất dấu tiếng Việt hoàn toàn và chỉ có **một** lỗi placeholder, nên rủi ro vỡ giao diện thấp; rủi ro chính là **mất uy tín sản phẩm** với người dùng Việt (`Quảng trường mô hình`, `Người mẫu hàng đầu`, `Nhận phòng hàng ngày`, `Cân bằng` cho *Balance*).

**`i18n/locales/vi.yaml` (backend) sạch.** 238/238 khóa khớp `en.yaml`, 0 lỗi placeholder, 0 chuỗi mất dấu, thuật ngữ nhất quán tuyệt đối (`hạn mức`, `token`, `kênh`, `mã đổi thưởng`, `upstream`, `mô hình`). **Không có mục sửa nào cho backend.** Bản backend chính là căn cứ để chọn thuật ngữ chuẩn ở mục 2.

---

## 2. Bảng thuật ngữ chuẩn

Chuẩn được chọn theo thứ tự ưu tiên: (1) cách dùng trong `i18n/locales/vi.yaml` đã được biên tập tốt, (2) tần suất áp đảo trong `vi.json`, (3) cách nói thực tế của lập trình viên Việt.

| Khái niệm (EN) | Chuẩn (VI) | Lý do |
|---|---|---|
| model | **mô hình** | 332/458 khóa đã dùng; backend dùng 100%. `mẫu` nghĩa là *sample/template*, `người mẫu` là *fashion model* — sai hẳn |
| Model Square (trang) | **Thư viện mô hình** | Đúng chức năng: `features/pricing/index.tsx` + nav `/pricing` là trang liệt kê mô hình & giá, không phải quảng trường |
| channel | **kênh** | 238 khóa; backend 100% |
| token (đơn vị tính phí & khóa) | **token** | Từ mượn lập trình viên Việt dùng; backend dùng 20/20 lần. `mã thông báo` là bản dịch Microsoft cho *auth token*, sai văn phong ở đây |
| API key | **khóa API** | 66 khóa (áp đảo); backend dùng `khóa` |
| group | **nhóm** | 249 khóa; backend 100% |
| quota | **hạn mức** | Backend dùng 13/13 lần; 57 khóa frontend. `hạn ngạch` là thuật ngữ hạn ngạch thương mại (quota xuất khẩu), sai ngữ cảnh |
| credit (đơn vị) | **credit** | Đơn vị tính phí, giữ nguyên như *token*. `tín dụng` gợi tín dụng ngân hàng |
| balance | **số dư** | 36 khóa dùng đúng; `cân bằng` là *equilibrium* |
| top-up / recharge | **nạp tiền** | 19 khóa; backend `topup.*` dùng "nạp tiền" |
| log | **nhật ký** | 55 khóa; backend 100%. Giữ `log` khi là tiền tố kỹ thuật (`log probabilities`) |
| redeem / redemption code | **mã đổi thưởng** | 39 khóa; backend 8/8 |
| upstream | **upstream** | Từ mượn; backend dùng 3/3 lần và không bao giờ dịch. `thượng nguồn` đúng nghĩa đen nhưng lạ tai trong ngữ cảnh hạ tầng |
| provider | **nhà cung cấp** | 102 khóa (áp đảo) |
| relay | **chuyển tiếp** | 34 khóa; giữ `Relay` khi là tên chỉ số/nhãn kỹ thuật |
| rate limit | **giới hạn tốc độ** | 12 khóa (áp đảo) |
| prompt / completion tokens | **token đầu vào / token đầu ra** | 6 khóa dùng "token đầu vào"; cặp vào/ra dễ hiểu hơn "token nhắc" |
| stream / streaming | **stream** (danh từ) / **phát trực tuyến** (động từ) | Hiện có 3 cách; chọn cặp này để phân biệt nhãn kỹ thuật với mô tả |
| tag | **thẻ** | 6/7 khóa đã dùng |
| check-in | **điểm danh** | 20/22 khóa đã dùng |
| new-api / New API / NewAPI / QuantumNous | **giữ nguyên** | Chính sách dự án (`AGENTS.md` § Protected project information) |

---

## 3. Danh sách sửa

Tất cả các hàng dưới đây thuộc `web/src/i18n/locales/vi.json`, đối tượng `translation`.
**Backend `i18n/locales/vi.yaml`: không có mục sửa.**

### 3.1 Hỏng placeholder / chuỗi bị cắt cụt (3) — SỬA TRƯỚC

| Khóa tiếng Anh | Giá trị VI hiện tại | Giá trị VI đề xuất | Lý do |
|---|---|---|---|
| `Minimum LinuxDO trust level required` | `Yêu cầu mức độ tin cậy {{LinuxDO}} tối thiểu` | `Yêu cầu mức độ tin cậy LinuxDO tối thiểu` | `{{LinuxDO}}` là placeholder i18next **bịa ra**; khóa gốc không có biến nào → i18next render ra chuỗi rỗng, mất chữ "LinuxDO" |
| `Select models (empty for allow all)` | `Chọn model (để trống nếu muốn cho` | `Chọn mô hình (để trống để cho phép tất cả)` | Chuỗi bị cắt cụt giữa câu, ngoặc không đóng |
| `Delete invalid codes (used/disabled/expired)` | `Xóa mã không hợp lệ (đã sử dụng/đã vô hiệu hóa/đ` | `Xóa mã không hợp lệ (đã dùng/đã tắt/đã hết hạn)` | Chuỗi bị cắt cụt giữa từ, ngoặc không đóng |

### 3.2 Vi phạm thương hiệu (6) — tên sản phẩm "New API" bị dịch

| Khóa tiếng Anh | Giá trị VI hiện tại | Giá trị VI đề xuất | Lý do |
|---|---|---|---|
| `New API` | `API mới` | `New API` | Tên sản phẩm được bảo vệ |
| `New API &lt;noreply@example.com&gt;` | `API mới &lt;noreply@example.com&gt;` | `New API &lt;noreply@example.com&gt;` | Tên sản phẩm được bảo vệ |
| `New API Project Repository:` | `Kho lưu trữ Dự án API Mới:` | `Kho mã dự án New API:` | Tên sản phẩm được bảo vệ |
| `Welcome to our New API...` | `Chào mừng bạn đến với API mới của chúng tôi...` | `Chào mừng bạn đến với New API...` | Tên sản phẩm được bảo vệ |
| `e.g. New API Console` | `Ví dụ: Bảng điều khiển API mới` | `Ví dụ: New API Console` | Tên sản phẩm được bảo vệ |
| `Warning: Base URL should not end with /v1. New API will handle it automatically. This may cause request failures.` | `Cảnh báo: URL cơ sở không nên kết thúc bằng /v1. API mới sẽ xử lý tự động. Điều này có thể gây ra lỗi yêu cầu.` | `Cảnh báo: URL cơ sở không nên kết thúc bằng /v1. New API sẽ xử lý tự động. Điều này có thể gây ra lỗi yêu cầu.` | Tên sản phẩm được bảo vệ |

> **Không phải lỗi:** `No proprietary SDK, no new API to learn.` → `...không phải học API mới.` Ở đây "a new API" là danh từ chung, dịch đúng. `NewAPI`, `QuantumNous`, URL GitHub đều giữ nguyên — đạt.

### 3.3 Dịch sai nghĩa / dịch máy nghĩa đen (20)

| Khóa tiếng Anh | Giá trị VI hiện tại | Giá trị VI đề xuất | Lý do |
|---|---|---|---|
| `Model Square` | `Quảng trường mô hình` | `Thư viện mô hình` | "Square" ở đây là *showcase*, không phải quảng trường thành phố. Ngữ cảnh: nav → `/pricing`, `features/pricing/index.tsx:182` |
| `Collect relay latency and success-rate metrics for the model square.` | `Thu thập độ trễ Relay và tỷ lệ thành công cho quảng trường mô hình.` | `Thu thập độ trễ relay và tỷ lệ thành công cho Thư viện mô hình.` | Cùng lỗi |
| `Top Models` | `Người mẫu hàng đầu` | `Mô hình hàng đầu` | "Người mẫu" = người mẫu thời trang. Ngữ cảnh: `features/rankings/components/models-section.tsx:173` |
| `Original Model` | `Nguyên mẫu` | `Mô hình gốc` | "Nguyên mẫu" = *prototype*; đây là cột model gốc trong bảng ánh xạ |
| `Model Name` | `Tên mẫu` | `Tên mô hình` | "mẫu" = sample/template |
| `Model Name *` | `Tên mẫu *` | `Tên mô hình *` | như trên |
| `Model name` | `Tên mẫu` | `Tên mô hình` | như trên |
| `Select Model` | `Chọn mẫu` | `Chọn mô hình` | như trên |
| `Search model name...` | `Tìm kiếm tên mẫu...` | `Tìm tên mô hình...` | như trên |
| `Filter by model...` | `Lọc theo mẫu...` | `Lọc theo mô hình...` | như trên |
| `All Models` | `Tất cả các mẫu` | `Tất cả mô hình` | như trên |
| `The sync will fetch missing models and vendors from the selected source. Existing records are updated only when you approve conflicts.` | `Đồng bộ hóa sẽ tìm nạp các mẫu và nhà cung cấp còn thiếu từ nguồn đã chọn. Các bản ghi hiện có chỉ được cập nhật khi bạn chấp thuận các xung đột.` | `Đồng bộ sẽ lấy các mô hình và nhà cung cấp còn thiếu từ nguồn đã chọn. Bản ghi hiện có chỉ được cập nhật khi bạn chấp thuận xung đột.` | như trên |
| `Balance` | `Cân bằng` | `Số dư` | "Cân bằng" = *equilibrium*. Ngữ cảnh: nhãn số dư trong `features/usage-logs/components/dialogs/user-info-dialog.tsx:118` |
| `Load Balancing` | `Tải cân bằng` | `Cân bằng tải` | Sai trật tự từ; các khóa khác đã dùng đúng "cân bằng tải" |
| `Check in daily to receive random quota rewards` | `Nhận phòng hàng ngày để nhận phần thưởng theo hạn ngạch ngẫu nhiên` | `Điểm danh hàng ngày để nhận thưởng hạn mức ngẫu nhiên` | "Nhận phòng" = check-in khách sạn |
| `Deterministic sampling seed (best-effort)` | `Hạt giống lấy mẫu xác định (cố gắng tốt nhất)` | `Seed lấy mẫu tất định (nỗ lực tối đa)` | "Hạt giống" là nghĩa đen thực vật; `seed` là thuật ngữ kỹ thuật giữ nguyên |
| `Discount ratio for cache hits.` | `Tỷ lệ chiết khấu cho lượt truy cập bộ nhớ đệm thành công.` | `Tỷ lệ chiết khấu khi trúng bộ nhớ đệm.` | *cache hit* = "trúng cache", không phải "lượt truy cập" |
| `Optional ratio used when upstream cache hits occur.` | `Tỷ lệ tùy chọn được sử dụng khi xảy ra các lượt truy cập bộ nhớ đệm ngược dòng.` | `Tỷ lệ tùy chọn áp dụng khi trúng bộ nhớ đệm ở upstream.` | Cùng lỗi + `ngược dòng` (xem 3.5) |
| `Add credits` | `Thêm tín dụng` | `Thêm credit` | `tín dụng` = tín dụng ngân hàng |
| `Credit remaining` | `Tín dụng còn lại` | `Credit còn lại` | như trên |

> Ngoài ra: `Create and review invite or credit codes.` → `Tạo và xem xét mã mời hoặc mã tín dụng.` nên đổi `mã tín dụng` → `mã credit` (đã tính trong cụm nhất quán 3.5).

### 3.4 Còn nguyên tiếng Anh (41)

#### 3.4.1 Chuỗi tiếng Anh nguyên vẹn hoặc chỉ diễn giải lại bằng tiếng Anh (30)

| Khóa tiếng Anh | Giá trị VI hiện tại | Giá trị VI đề xuất |
|---|---|---|
| `Amount to pay:` | `Amount due:` | `Số tiền cần thanh toán:` |
| `Append to existing keys` | `Add to existing keys` | `Thêm vào các khóa hiện có` |
| `Blocked keywords` | `Blocked keyword` | `Từ khóa bị chặn` |
| `Clear filters` | `Clear filter` | `Xóa bộ lọc` |
| `Current Value` | `Present value` | `Giá trị hiện tại` |
| `Date and time when this announcement should be displayed` | `The date and time this notification should be displayed` | `Ngày giờ hiển thị thông báo này` |
| `Enter new tag name or leave empty` | `Enter new tag name or leave blank` | `Nhập tên thẻ mới hoặc để trống` |
| `Enterprise Account` | `Business account` | `Tài khoản doanh nghiệp` |
| `General Settings` | `General settings` | `Cài đặt chung` |
| `Invitee Reward` | `Referral reward` | `Thưởng cho người được mời` |
| `Max Requests (incl. failures)` | `Maximum number of requests (including errors)` | `Số yêu cầu tối đa (tính cả lỗi)` |
| `Model fixed pricing` | `Fixed-price model` | `Giá cố định theo mô hình` |
| `Official Sync` | `Official sync` | `Đồng bộ chính thức` |
| `Oops! Something went wrong` | `Oops! An error occurred.` | `Rất tiếc! Đã xảy ra lỗi` |
| `Pricing Configuration` | `Price configuration` | `Cấu hình giá` |
| `Pricing Type` | `Price type` | `Loại giá` |
| `Query Param` | `Query param` | `Tham số truy vấn` |
| `Quota given to users who invite others` | `Limit for users inviting others` | `Hạn mức thưởng cho người mời` |
| `Rate Limiting` | `Rate limit` | `Giới hạn tốc độ` |
| `Replacement Model` | `Replacement model` | `Mô hình thay thế` |
| `Request Count` | `Number of requests` | `Số yêu cầu` |
| `Reveal key` | `Display key` | `Hiện khóa` |
| `Search groups...` | `Searching for group...` | `Tìm nhóm...` |
| `Select announcement type` | `Select notification type` | `Chọn loại thông báo` |
| `Select granularity` | `Select detail level` | `Chọn độ chi tiết` |
| `Sync this model with official upstream` | `Synchronize this model with the official source.` | `Đồng bộ mô hình này với upstream chính thức` |
| `Target group` | `Target audience` | `Nhóm mục tiêu` |
| `The token group that will have a custom ratio` | `The token group will have a custom ratio.` | `Nhóm token sẽ áp dụng tỷ lệ tùy chỉnh` |
| `Total Earned` | `Total income` | `Tổng thu nhập` |
| `Upstream Response` | `Upstream feedback` | `Phản hồi từ upstream` |
| `View Pricing` | `View price` | `Xem bảng giá` |

#### 3.4.2 Nhãn/từ đơn chưa dịch (11)

| Khóa tiếng Anh | Giá trị VI hiện tại | Giá trị VI đề xuất | Ngữ cảnh đã kiểm |
|---|---|---|---|
| `Add` | `Add` | `Thêm` | nút, `announcements-section.tsx:461` |
| `All` | `All` | `Tất cả` | tab, `usage-logs/index.tsx:135` |
| `Asc` | `Asc` | `Tăng dần` | sắp xếp, `data-table/core/column-header.tsx:78` (cặp `Desc` đã là `Giảm dần`) |
| `End` | `End` | `Kết thúc` | nhãn ngày, `user-subscriptions-dialog.tsx:335` |
| `Tag` | `Tag` | `Thẻ` | tiêu đề cột, `channels-columns.tsx:1100` (các khóa `Tag *` khác đã là `thẻ`) |
| `Pay` | `Pay` | `Thanh toán` | nút nạp tiền, `recharge-form-card.tsx:223` |
| `and` | `and` | `và` | `auth/components/terms-footer.tsx:82` |
| `of` | `of` | `trên` | "Showing 1-10 **of** 100", `missing-models-dialog.tsx:142` |
| `of 3:` | `of 3:` | `trên 3:` | "Bước 2 **of 3:**", `two-fa-setup-dialog.tsx:142` |
| `off` | `off` | `giảm` | "20% **off**", `amount-discount-visual-editor.tsx:176` |
| `{{count}} days remaining` | `{{count}} days remaining` | `Còn {{count}} ngày` | `subscription-plans-card.tsx:449` |

> **Không phải lỗi (giữ nguyên):** `API`, `URL`, `JSON`, `Email`, `Webhook`, `Endpoint`, `Header`, `Boolean`, `Seed`, `Video`, `Plugin`, `Slug`, `OAuth`, `OIDC`, `TTL`, `RPM`, `TPM`, `Root` (tên vai trò), `Submodel` (tên nhà cung cấp kênh #53), `Inpaint`/`Pan`/`Zoom` (tên thao tác Midjourney), `Embeddings`, `credit`, `token`, mọi tên hãng và mọi URL ví dụ.

### 3.5 Thuật ngữ không nhất quán (92)

Các cụm dưới đây áp dụng được bằng tìm–thay chuỗi con **trong đúng danh sách khóa được liệt kê** (không thay toàn file để tránh chạm các khóa khác).

#### a) `token` — 17 khóa dùng `mã thông báo`

Quy tắc: `Mã thông báo` → `Token`, `mã thông báo` → `token`.

| Khóa | Hiện tại | Đề xuất |
|---|---|---|
| `Token` | `Mã thông báo` | `Token` |
| `Tokens` | `Mã thông báo` | `Token` |
| `tokens` | `mã thông báo` | `token` |
| `Token Name` | `Tên mã thông báo` | `Tên token` |
| `Tokens Only` | `Chỉ mã thông báo` | `Chỉ token` |
| `Copy token` | `Sao chép mã thông báo` | `Sao chép token` |
| `No token found.` | `Không tìm thấy mã thông báo.` | `Không tìm thấy token.` |
| `Enter new token to update` | `Nhập mã thông báo mới để cập nhật` | `Nhập token mới để cập nhật` |
| `Bot Token` | `Mã thông báo Bot` | `Token bot` |
| `Server Token` | `Mã thông báo máy chủ` | `Token máy chủ` |
| `Gotify Application Token` | `Mã thông báo ứng dụng Gotify` | `Token ứng dụng Gotify` |
| `Your Telegram Bot Token` | `Mã thông báo bot Telegram của bạn` | `Token bot Telegram của bạn` |
| `Token obtained from your Gotify application` | `Mã thông báo thu được từ ứng dụng Gotify của bạn` | `Token lấy từ ứng dụng Gotify của bạn` |
| `3. Enter your Gotify server URL and token above` | `3. Nhập URL máy chủ Gotify và mã thông báo của bạn ở trên` | `3. Nhập URL máy chủ Gotify và token của bạn ở trên` |
| `Generate and manage your API access token` | `Tạo và quản lý mã thông báo truy cập API của bạn` | `Tạo và quản lý token truy cập API của bạn` |
| `Statistical tokens` | `Mã thông báo thống kê` | `Token thống kê` |
| `Budget Tokens Ratio` | `Tỷ lệ Mã thông báo Ngân sách` | `Tỷ lệ token ngân sách` |

#### b) `quota` — 25 khóa dùng `hạn ngạch`, 6 khóa để nguyên `quota`

Quy tắc cho nhóm 1: `Hạn ngạch` → `Hạn mức`, `hạn ngạch` → `hạn mức`.

Danh sách khóa nhóm 1 (25):
`Allow users to check in daily for random quota rewards`, `Allow wallet balance after quota used up`, `By quota`, `Check in daily to receive random quota rewards`, `Choose how quota values are shown to users`, `Enter a positive or negative amount to adjust the quota`, `Enter the quota amount in tokens`, `Maximum check-in quota`, `Maximum quota amount awarded for check-in`, `Minimum check-in quota`, `Minimum quota amount awarded for check-in`, `No Quota`, `Plan Quota`, `Quota`, `Quota Distribution`, `Quota Types`, `Quota clamped`, `Quota saturation protection triggered`, `Quota:`, `Remaining quota`, `Remaining quota units`, `Show prices in currency instead of quota.`, `Total consumed quota`, `Total quota included in the plan, usable per billing period. 0 means unlimited.`, `User dashboard and quota controls.`

Nhóm 2 — `quota` để nguyên tiếng Anh (6), sửa từng mục:

| Khóa | Hiện tại | Đề xuất |
|---|---|---|
| `Amount of quota to credit to user account.` | `Số lượng quota để ghi có vào tài khoản người dùng.` | `Số hạn mức cộng vào tài khoản người dùng.` |
| `Tokens-only mode will show raw quota values regardless of this toggle.` | `Chế độ Tokens-only sẽ hiển thị giá trị quota thô bất kể tùy chọn này.` | `Chế độ chỉ-token sẽ hiển thị giá trị hạn mức thô bất kể tùy chọn này.` |
| `Quota lands in your account` | `Quota được cộng vào tài khoản` | `Hạn mức được cộng vào tài khoản` |
| `GPT, Claude, Gemini, DeepSeek, Qwen and 40+ other providers — ... quota is added automatically.` | `... SePay, quota được cộng tự động.` | `... SePay, hạn mức được cộng tự động.` |
| `No Visa or Mastercard, no foreign currency wallet needed. ... adds quota automatically.` | `... SePay đối soát và cộng quota tự động.` | `... SePay đối soát và cộng hạn mức tự động.` |
| `Quota Per Unit` | `Định mức mỗi đơn vị` | `Hạn mức mỗi đơn vị` |

#### c) `upstream` — 39 khóa dùng `thượng nguồn`, 2 khóa dùng `ngược dòng`

Quy tắc: `Thượng nguồn` → `Upstream`, `thượng nguồn` → `upstream`, `ngược dòng` → `upstream`. Sau khi thay, rà lại trật tự từ tiếng Việt (ví dụ `nhà cung cấp thượng nguồn` → `nhà cung cấp upstream`, đúng ngữ pháp; `Thượng nguồn tương thích OpenAI` → `Upstream tương thích OpenAI`).

Danh sách khóa (39): `All upstream data is trusted`, `Applied upstream model changes to channel (ID: {{id}})`, `Applied upstream model changes to {{count}} channels`, `Batch upstream model update`, `Choose a complete upstream protocol plan or edit individual route groups.`, `Choose where to fetch upstream metadata.`, `Deploy your own gateway and start routing requests through your configured upstream services.`, `Fetching upstream ratios...`, `Format: AccessKey|SecretKey (or just ApiKey if upstream is New API)`, `High-risk status code retry risk check 2`, `High-risk status code retry risk disclaimer`, `If an upstream error contains any of these keywords (case insensitive), the channel will be disabled automatically.`, `No upstream ratio differences found`, `Open Query Balance to view the upstream JSON response`, `OpenAI Compatible Upstream`, `OpenAI Models route is required to enable upstream model checks`, `OpenAI Models upstream path must not contain {model}`, `Ratio applied to audio inputs where supported by the upstream model.`, `Recommended to keep this high to avoid upstream throttling.`, `Select the fields you want to overwrite with upstream data. Unselected fields keep their local values.`, `Synced upstream models`, `Synchronize models and vendors from an upstream source`, `The mapped upstream model(s)`, `The upstream channel that served the requests`, `The upstream natively supports all three protocols; every selected route is forwarded without conversion.`, `The upstream response is valid JSON, but it does not match the OpenAI credit_summary format. The channel balance was not updated.`, `This route discovers upstream OpenAI models and cannot be split or matched by client model rules.`, `This route is used only by channel management to discover upstream models.`, `This route is used only by channel management to query the upstream balance.`, `Uploading a plugin is an administrator-level trust decision. A plugin can access channel credentials and shape upstream requests. Review its source and diff before activation.`, `Upstream`, `Upstream JSON response`, `Upstream Request ID`, `Upstream Task ID`, `Upstream model detection task started. Track progress in System Info, then refresh to review staged updates.`, `Upstream price sync`, `Upstream protocol plan`, `Users call the model on the left. The platform forwards the request to the upstream model on the right.`, `upstream services integrated`

Hai khóa dùng `ngược dòng`:

| Khóa | Hiện tại | Đề xuất |
|---|---|---|
| `Forward requests directly to upstream providers without any post-processing.` | `Chuyển tiếp các yêu cầu trực tiếp đến các nhà cung cấp ngược dòng mà không cần xử lý hậu kỳ nào.` | `Chuyển tiếp yêu cầu trực tiếp đến nhà cung cấp upstream, không xử lý thêm.` (`xử lý hậu kỳ` là thuật ngữ dựng phim) |
| `Optional ratio used when upstream cache hits occur.` | (xem 3.3) | (xem 3.3) |

#### d) `model` để nguyên tiếng Anh giữa câu tiếng Việt (5 khóa rõ ràng)

| Khóa | Hiện tại | Đề xuất |
|---|---|---|
| `Model deleted successfully` | `Model đã được xóa thành công` | `Đã xóa mô hình thành công` |
| `Client model matching` | `Khớp client model` | `Khớp mô hình phía client` |
| `Client model matching help` | `Trợ giúp khớp client model` | `Trợ giúp khớp mô hình phía client` |
| `Model regex cannot be empty` | `Regex model không được để trống` | `Regex mô hình không được để trống` |
| `{{n}} model(s) selected` | `Đã chọn {{n}} model` | `Đã chọn {{n}} mô hình` |

> Các khóa mô tả route (`Use exact client model names...`, `Rules match the original model value...`, v.v.) cố tình giữ `model` vì đang nói về **trường JSON tên `model`** trong body yêu cầu — không phải lỗi.

#### e) Cụm nhỏ còn lại (6)

| Khóa | Hiện tại | Đề xuất | Lý do |
|---|---|---|---|
| `Recharge` | `Nạp lại` | `Nạp tiền` | Đồng bộ với `Top-up` → `Nạp tiền`; "nạp lại" gợi sạc pin |
| `Online topup is not enabled. Please use redemption code or contact administrator.` | `...Vui lòng sử dụng mã quy đổi...` | `...Vui lòng sử dụng mã đổi thưởng...` | 39/40 khóa dùng `mã đổi thưởng` |
| `Group-based rate limits` | `Giới hạn tỷ lệ dựa trên nhóm` | `Giới hạn tốc độ theo nhóm` | `giới hạn tỷ lệ` sai nghĩa |
| `Redis-backed caching with per-IP and per-key rate limiting` | `Cache Redis, giới hạn tần suất theo IP và theo key` | `Cache Redis, giới hạn tốc độ theo IP và theo khóa` | Đồng bộ thuật ngữ |
| `Streaming` | `Truyền liên tục` | `Phát trực tuyến` | 3 cách dịch cho cùng khái niệm |
| `Non-stream` | `Không phát trực tuyến` | `Không stream` | Cặp với `Stream` → `Stream` (nhãn bộ lọc nhật ký) |
| `Create and review invite or credit codes.` | `Tạo và xem xét mã mời hoặc mã tín dụng.` | `Tạo và xem mã mời hoặc mã credit.` | Đồng bộ `credit` |
| `Do not repeat check-in; only once per day` | `Không lặp lại check-in; chỉ một lần mỗi ngày` | `Không điểm danh lặp lại; chỉ một lần mỗi ngày` | 20/22 khóa dùng `điểm danh` |
| `No group-based rate limits configured. Click "Add group" to get started.` | `Chưa cấu hình giới hạn tốc độ dựa trên nhóm. Nhấp "Add group" để bắt đầu.` | `Chưa cấu hình giới hạn tốc độ theo nhóm. Nhấn "Thêm nhóm" để bắt đầu.` | Nhãn nút trích dẫn phải khớp nhãn nút thật (`Add group` → `Thêm nhóm`) |

### 3.6 Ngữ pháp / văn phong / dấu (17)

#### a) Viết hoa kiểu tiếng Anh (Title Case) — tiếng Việt dùng viết hoa đầu câu (12)

| Khóa | Hiện tại | Đề xuất |
|---|---|---|
| `Current Balance` | `Số Dư Hiện Tại` | `Số dư hiện tại` |
| `Create API Key` | `Tạo Khóa API` | `Tạo khóa API` |
| `Create Prefill Group` | `Tạo Nhóm Điền Sẵn` | `Tạo nhóm điền sẵn` |
| `Add Uptime Kuma Group` | `Thêm Nhóm Uptime Kuma` | `Thêm nhóm Uptime Kuma` |
| `Add Group` | `Thêm Nhóm` | `Thêm nhóm` |
| `Delete All Disabled` | `Xóa Tất Cả Đã Tắt` | `Xóa tất cả mục đã tắt` |
| `Display Name Field` | `Trường Tên Hiển Thị` | `Trường tên hiển thị` |
| `Edit OAuth Provider` | `Chỉnh Sửa Nhà Cung Cấp OAuth` | `Chỉnh sửa nhà cung cấp OAuth` |
| `Field Mapping` | `Ánh Xạ Trường` | `Ánh xạ trường` |
| `Fill All Models` | `Điền Tất Cả Mô Hình` | `Điền tất cả mô hình` |
| `Select Sync Source` | `Chọn Nguồn Đồng Bộ` | `Chọn nguồn đồng bộ` |
| `footer.columns.docs.links.installation` | `Hướng Dẫn Cài Đặt` | `Hướng dẫn cài đặt` |

> Cùng lỗi ở `footer.columns.docs.links.quickStart` (`Bắt Đầu Nhanh` → `Bắt đầu nhanh`) và `footer.columns.related.title` (`Các Dự Án Liên Quan` → `Các dự án liên quan`) — đã tính trong tổng.

#### b) Hậu tố số nhiều `(s)` của tiếng Anh dính vào tiếng Việt (4)

Tiếng Việt không biến hình số nhiều; `(s)` hiển thị như lỗi.

| Khóa | Hiện tại | Đề xuất |
|---|---|---|
| `The mapped upstream model(s)` | `Mô hình(s) thượng nguồn được ánh xạ` | `Các mô hình upstream được ánh xạ` |
| `channel(s)? This action cannot be undone.` | `kênh(s)? Hành động này không thể hoàn tác.` | `kênh? Hành động này không thể hoàn tác.` |
| `model(s)? This action cannot be undone.` | `mô hình(s)? Hành động này không thể hoàn tác.` | `mô hình? Hành động này không thể hoàn tác.` |
| `model(s) selected out of` | `mô hình(s) được chọn trong số` | `mô hình được chọn trong số` |

#### c) Chính tả dấu không thống nhất (1)

| Khóa | Hiện tại | Đề xuất | Lý do |
|---|---|---|---|
| `Set this value in your SePay dashboard. Leave blank unless rotating the key.` | `...nếu không đổi khoá.` | `...nếu không đổi khóa.` | 66 khóa khác dùng `khóa` (kiểu đặt dấu hiện đại); đây là mục duy nhất dùng `khoá` |

> **Không phải lỗi:** không có mục nào trong `vi.json` hay `vi.yaml` bị mất dấu tiếng Việt (ASCII-stripped). Kiểm tra tự động toàn bộ 5.583 + 238 mục — sạch.

---

## 4. Câu hỏi chưa giải đáp

| # | Vấn đề | Cần quyết định |
|---|---|---|
| 1 | **`upstream` → giữ nguyên hay `thượng nguồn`?** Backend đã chọn giữ nguyên (3/3); frontend đang 48 giữ nguyên / 39 dịch. Tôi đề xuất giữ nguyên để khớp backend, nhưng đây là 39 chuỗi phải sửa. Nếu sản phẩm ưu tiên "tiếng Việt tối đa", hướng ngược lại (sửa 48 chuỗi frontend + 3 chuỗi backend) cũng nhất quán. | Chọn một hướng trước khi áp dụng 3.5(c) |
| 2 | **`Model Square` → tên trang.** Tôi đề xuất `Thư viện mô hình`. Các lựa chọn khác hợp lý: `Kho mô hình`, `Danh mục mô hình`, `Bảng giá mô hình` (trang thực tế là `/pricing`). Đây là nhãn điều hướng chính, nên là quyết định thương hiệu. | Chốt tên hiển thị |
| 3 | **`credit` vs `hạn mức`.** Sản phẩm dùng song song *quota* và *credit*; hiện `credit` bị dịch thành `tín dụng` (sai) nhưng cũng có nơi để nguyên. Nếu trong sản phẩm *credit* và *quota* là **cùng một thứ**, nên gộp về `hạn mức` thay vì giữ hai từ. | Xác nhận hai khái niệm có tách biệt về nghiệp vụ không |
| 4 | **Xưng hô.** Bản dịch hiện dùng "bạn" nhất quán ở câu dài nhưng lược bỏ ở nhãn ngắn. Với thị trường Việt B2B/developer, "bạn" là hợp lý; chỉ cần xác nhận không đổi sang "quý khách". | Xác nhận giữ "bạn" |
| 5 | **`Query`, `Store`, `Underground`, `preset.underground` (`Bóng đêm`), `Forest Whisper` (`Tiếng thì thầm rừng cây`).** Không tìm thấy nơi gọi `t('Query')`/`t('Store')` trong `web/src/`; nhóm `preset.*` là tên chủ đề giao diện — dịch thoáng là chủ ý hay lỗi thì cần chủ sản phẩm xác nhận. | Xác nhận nhóm `preset.*` / tên chủ đề có được dịch hay giữ tên gốc |

---

## 5. Điều KHÔNG được kiểm tra hết

- ~5.130 mục trong `vi.json` đã có dấu, không trùng khóa tiếng Anh, không trúng bộ lọc thuật ngữ nào và **chưa được đọc từng dòng**. Lỗi sắc thái/văn phong trong nhóm này có thể còn sót.
- Không kiểm tra độ dài chuỗi so với không gian giao diện (tiếng Việt dài hơn tiếng Anh ~15–25%); có thể có tràn nhãn nút chưa phát hiện.
- Không chạy ứng dụng để xác minh trực quan.
