# Phân tích khoảng trống tính năng cho thị trường Việt Nam — fork `new-api`

**Ngày**: 2026-09-05
**Phạm vi**: `D:/Dev/www/api-gateway`, so với upstream `QuantumNous/new-api` (commit nền `32c261923`, 2026-09-03) và nhu cầu thị trường VN.
**Phương pháp**: đọc mã nguồn/spec trong repo trước, sau đó tra cứu web (nguồn chính thức ưu tiên). Mọi khẳng định bên ngoài đều có URL + ngày truy cập. Những gì không xác minh được thì ghi rõ là "chưa xác minh".

---

## (a) Tóm tắt

1. Fork bám sát upstream: chỉ lệch 2 commit tính năng phía trên nền (`3a9f41ee8`, `7c044d7c5`, cả hai đều không liên quan VN) — trục "thiếu tính năng upstream" gần như trống, đúng như giả định.
2. SePay là cổng thanh toán duy nhất, và đó là **lựa chọn thực dụng có chủ đích**, không phải thiếu sót: MoMo/VNPay/ZaloPay đều bắt buộc đăng ký hộ kinh doanh/doanh nghiệp, SePay chỉ cần tài khoản ngân hàng cá nhân.
3. Khoảng trống lớn nhất về tuân thủ là **hóa đơn điện tử (VAT invoice)** — repo không có khái niệm hóa đơn nào, trong khi doanh nghiệp VN mua API credit thường cần hóa đơn hợp lệ theo Nghị định 70/2025 (sắp bị thay bởi Nghị định 254/2026 từ 01/07/2026).
4. Tài liệu vận hành SePay (`docs/payments-sepay.md`) — vốn là tính năng riêng của fork cho thị trường VN — lại được viết **bằng tiếng Trung**, không phải tiếng Việt. Toàn bộ `docs/` không có bản tiếng Việt nào.
5. Không có adapter cho nhà cung cấp AI Việt Nam (FPT, Viettel AI, GreenNode/VNG) dù các bên này đã có endpoint OpenAI-compatible; đây là gap dễ làm, effort thấp.
6. Không có Zalo OAuth, không có kênh thông báo Zalo ZNS — trong khi hệ thống oauth và notify đã có khung đăng ký sẵn (registry pattern), effort thêm 1 provider là vừa phải.
7. Hiển thị giá bằng VND trên trang pricing **đã khả thi bằng cấu hình admin** (`quota_display_type = CUSTOM` + `custom_currency_symbol = "₫"` + `custom_currency_exchange_rate`), không cần code — đây là phát hiện quan trọng làm giảm độ ưu tiên của mục "giá VND".
8. `docker-compose*.yml` mặc định `TZ=Asia/Shanghai` — lỗi nhỏ, chỉ ảnh hưởng hiển thị giờ trong log/container, không ảnh hưởng độ trễ mạng.
9. Không có xác thực số điện thoại (chỉ có xác thực email), không có khái niệm thuế thu nhập cá nhân/thuế trên nạp tiền trong hệ thống — cần làm rõ với chủ vận hành xem có bắt buộc theo quy định thuế hộ kinh doanh cá nhân hay không.
10. Đề xuất 2 việc làm trước: (1) viết tài liệu vận hành SePay bằng tiếng Việt (S, đã có sẵn tài liệu tiếng Trung để dịch), (2) làm rõ với chủ vận hành liệu có bắt buộc xuất hóa đơn VAT ngay bây giờ hay chưa (quyết định kinh doanh, không phải code).

---

## (b) Bảng ưu tiên

| # | Gap | Bằng chứng | Công sức | Rủi ro nếu bỏ qua |
|---|-----|------------|----------|---------------------|
| 1 | Tài liệu vận hành/người dùng tiếng Việt | `docs/payments-sepay.md` viết bằng tiếng Trung; `docs/translation-glossary.{fr,ru}.md` tồn tại nhưng không có bản `.vi.md`; `docs_link` mặc định trỏ `docs.newapi.pro` (Anh/Trung) | **S** — dịch tài liệu đã có, không cần code | Thấp kỹ thuật, cao vận hành: operator/khách hàng VN không tự tra cứu được cách cấu hình SePay, webhook, xoay khóa API |
| 2 | Hóa đơn điện tử / VAT | Không tìm thấy `invoice`, `hoa_don`, `vat` trong `controller/`, `model/`, `web/src/features/wallet` | **M–L** — cần chọn nhà cung cấp e-invoice, tích hợp API, lưu trạng thái phát hành, UI yêu cầu xuất hóa đơn | Trung bình–cao nếu bán cho doanh nghiệp/kế toán VN; thấp nếu khách hàng chủ yếu là cá nhân/dev |
| 3 | Adapter model VN/SEA (GreenNode, FPT AI) | 118 hằng số kênh trong `constant/channel.go`, 45 thư mục `relay/channel/`, không có mục nào cho FPT/Viettel/GreenNode/VNG | **S–M** — hầu hết đã OpenAI-compatible nên có thể dùng lại adapter `openai/` qua custom base URL, không nhất thiết cần adapter riêng | Thấp — đã có đường vòng dùng "custom channel" trỏ base URL |
| 4 | Zalo OAuth (đăng nhập) | `oauth/registry.go` có pattern `Register()`; `oauth/github.go` (183 dòng) làm mẫu; không có `oauth/zalo.go` | **M** — 1 provider mới theo khuôn có sẵn, nhưng Zalo OAuth v3 cần app được Zalo duyệt và có App ID/Secret riêng | Thấp–trung bình — nhiều người dùng VN quen đăng nhập Zalo hơn GitHub, nhưng Google/GitHub đã đủ dùng cho đối tượng dev |
| 5 | Zalo ZNS (thông báo) | `service/user_notify.go` đã có 4 kênh: Email, Webhook, Bark, Gotify — theo `switch notifyType`; không có Zalo | **M** — thêm 1 case trong switch + tích hợp API ZNS (cần Zalo OA đã duyệt + trả phí/tin nhắn) | Thấp — Email/Webhook đã đủ cho use case kỹ thuật (dev dùng API), ZNS có giá trị hơn cho thông báo giao dịch/marketing |
| 6 | Hiển thị giá VND trên trang pricing | `quota_display_type` đã có `CUSTOM` (`setting/operation_setting/general_setting.go`), `web/src/lib/currency.ts` xử lý case `CUSTOM` với `customCurrencySymbol`/`customCurrencyExchangeRate`; `pricing-preview.tsx` và trang pricing đều dùng chung store này | **Không cần code** — chỉ cần admin cấu hình `quota_display_type=CUSTOM`, `custom_currency_symbol=₫`, `custom_currency_exchange_rate=<tỷ giá>` | Không có — đây là gap giả, chỉ cần hướng dẫn vận hành |
| 7 | Vùng/độ trễ (region) | `docker-compose.yml`, `docker-compose.dev.yml`: `TZ=Asia/Shanghai` mặc định | **S** — đổi default sang `Asia/Ho_Chi_Minh` trong file compose mẫu | Rất thấp — chỉ ảnh hưởng nhãn thời gian hiển thị, Singapore vẫn là hạ tầng hợp lý (10–40ms tới VN) |
| 8 | Xác thực số điện thoại | Không có trường `Phone` trong `model/user.go`; chỉ có `EmailVerificationEnabled` | Không đề xuất làm ngay — **N/A, cần làm rõ nhu cầu** | Thấp — không có bằng chứng đây là rào cản mua hàng với đối tượng dev/API |
| 9 | Thuế trên nạp tiền/thu nhập | Không tìm thấy logic thuế nào trong `controller/topup.go`, `controller/sepay.go` | Không đề xuất làm ngay — câu hỏi pháp lý/kế toán, không phải thiếu tính năng | Cần xác nhận với kế toán/luật sư của operator, ngoài phạm vi kỹ thuật |
| 10 | Fork lệch 2 commit upstream không-VN | `7c044d7c5` (2026-09-04, model modifiers/billing identity), `3a9f41ee8` (2026-09-04, tắt tạm `/messages/count_tokens`) | **S** nếu muốn đồng bộ — không phải gap thị trường VN | Rất thấp, chỉ là bảo trì thường lệ |

---

## (c) Chi tiết từng gap

### 1. Độ phủ thanh toán nội địa (MoMo, ZaloPay, VNPay, VietQR/NAPAS trực tiếp)

**Hiện trạng repo**: SePay là cổng duy nhất (`controller/sepay.go`, `controller/topup.go`, `controller/payment_compliance.go`, `setting/payment_sepay.go`). `proposal.md` của change `2026-09-04-replace-payment-gateways-with-sepay` nêu rõ lý do: 5 cổng cũ (Epay, Stripe, Creem, Waffo, Waffo Pancake) đều hướng thẻ/checkout quốc tế, không ai trong số operator hiện tại dùng. `design.md` xác nhận đây là quyết định có chủ đích ("Non-Goals: No pluggable payment-gateway abstraction. One provider does not justify an interface"), chấp nhận rủi ro "Single point of failure: one gateway" và có kế hoạch dự phòng là redemption code + hoàn tất thủ công qua admin.

**Bằng chứng thị trường**: MoMo, ZaloPay, VNPay đều yêu cầu hồ sơ đăng ký merchant gồm Giấy phép đăng ký kinh doanh (doanh nghiệp) hoặc Giấy chứng nhận đăng ký hộ kinh doanh — cá nhân không đăng ký kinh doanh không thể trở thành đơn vị chấp nhận thanh toán trực tiếp của các ví/cổng này (nguồn: hướng dẫn đăng ký VNPAY-QR, truy cập 2026-09-05, https://vnpay.vn/dang-ky-vnpay-qr-cho-cua-hang-00izi3uzy80zq). Từ 01/03/2026, Thông tư 25/2025/TT-NHNN còn yêu cầu hộ kinh doanh dùng tài khoản ngân hàng đứng tên hộ kinh doanh cho hoạt động kinh doanh (nguồn: cùng trang, truy cập 2026-09-05).

SePay ngược lại hỗ trợ cả tài khoản ngân hàng cá nhân lẫn tổ chức, chỉ cần đăng ký tài khoản SePay (không phải merchant account với ngân hàng/ví), kết nối qua API ngân hàng hoặc SMS biến động số dư (nguồn: SePay docs "Hướng dẫn thêm tài khoản ngân hàng", truy cập 2026-09-05, https://docs.sepay.vn/them-tai-khoan-ngan-hang.html). VietQR/NAPAS là chuẩn QR liên ngân hàng mà chính SePay/VNPay đều dùng làm lớp hiển thị — không phải một cổng thanh toán độc lập cần tích hợp riêng (nguồn: Kaadxpay "Vietnam Payments Guide: NAPAS, VietQR, MoMo & SBV Realities 2026", truy cập 2026-09-05, https://www.kaadxpay.com/en/countries/vietnam).

**Kết luận**: SePay-only **là lựa chọn đúng cho một operator cá nhân/nhỏ chưa đăng ký kinh doanh**, không phải khoảng trống cần lấp ngay. Việc thêm MoMo/ZaloPay/VNPay chỉ có ý nghĩa **khi operator đã có pháp nhân/hộ kinh doanh đăng ký** — lúc đó lợi ích chính là mở thêm kênh nạp tiền cho người dùng không quen chuyển khoản ngân hàng (ví điện tử). Đây là quyết định kinh doanh, nên hỏi lại chủ dự án trước khi lên kế hoạch — không tự ý đảo ngược quyết định đã ghi trong `design.md` (Non-Goals).

**Effort nếu triển khai sau này**: L — cần trừu tượng hóa gateway (điều mà `design.md` D-non-goal cố ý tránh cho 1 provider), cộng thủ tục pháp lý đăng ký merchant.

### 2. Hóa đơn điện tử (VAT / hóa đơn)

**Hiện trạng repo**: Không có bảng, controller, hay UI nào liên quan hóa đơn. `grep -rniE "invoice|hoa.?don|vat\b"` trên `controller/`, `model/`, `web/src/features/wallet`, `web/src/features/pricing` chỉ khớp một chuỗi tiếng Anh không liên quan ("One combined invoice" trong `web/src/features/home/constants.ts`, phần nói về mô hình kinh doanh chung, không phải tính năng xuất hóa đơn).

**Khung pháp lý hiện hành và sắp tới**:
- Nghị định 123/2020/NĐ-CP (hóa đơn, chứng từ) đã được sửa đổi bởi Nghị định 70/2025/NĐ-CP, hiệu lực từ 01/06/2025 — mở rộng đối tượng được ủy nhiệm lập hóa đơn điện tử ra cả hộ kinh doanh, cá nhân kinh doanh (nguồn: LuatVietnam, truy cập 2026-09-05, https://luatvietnam.vn/tin-van-ban-moi/nghi-dinh-123-2020-nd-cp-va-nghi-dinh-70-2025-nd-cp-ve-hoa-don-chinh-thuc-het-hieu-luc-tu-01-7-2026-186-110076-article.html).
- Thông tư 78/2021/TT-BTC đã được thay thế bởi Thông tư 32/2025/TT-BTC (nguồn: cùng bài LuatVietnam).
- Từ 01/07/2026, toàn bộ khung Nghị định 123/2020 sẽ hết hiệu lực, thay bằng **Nghị định 254/2026/NĐ-CP** cùng Thông tư 91/2026/TT-BTC, theo Luật Quản lý thuế 2025 mới (nguồn: LuatVietnam, truy cập 2026-09-05; MISA meInvoice công bố đã đáp ứng Nghị định 254/2026 và Thông tư 91/2026, truy cập 2026-09-05, https://www.misa.vn/154989/tai-lieu-open-api-tich-hop-hoa-don-dien-tu-misa-meinvoice-dau-ra/).
- **Chưa xác minh** được toàn văn nội dung Nghị định 254/2026 (chỉ có mô tả gián tiếp qua các trang tổng hợp luật, không truy cập được văn bản gốc trong phiên nghiên cứu này).

**Nhà cung cấp API hóa đơn điện tử**: MISA meInvoice có Open API tích hợp hóa đơn đầu ra công khai tài liệu (truy cập 2026-09-05, https://www.misa.vn/154989/tai-lieu-open-api-tich-hop-hoa-don-dien-tu-misa-meinvoice-dau-ra/, giá từ 300đ/hóa đơn theo trang chủ MISA). Viettel SInvoice và VNPT Invoice cũng có tích hợp qua các nền tảng bán hàng (ví dụ KiotViet) nhưng bài tìm được không trực tiếp là tài liệu API chính thức của Viettel/VNPT — **cần tra cứu thêm developer portal của Viettel S-Invoice và VNPT Invoice riêng để có thông tin API chính xác, chưa xác minh sâu trong phiên này**. BKAV eHoadon cũng xuất hiện trong kết quả tìm kiếm nhưng chỉ ở dạng trang so sánh nghị định, chưa xác minh về API.

**Mức tối thiểu để tuân thủ khi cần**: (1) thu thập mã số thuế/tên đơn vị người mua tại thời điểm nạp tiền (tùy chọn, không bắt buộc mọi giao dịch), (2) gọi API một nhà cung cấp hóa đơn điện tử đã được Tổng cục Thuế công nhận để phát hành hóa đơn khi khách yêu cầu, (3) lưu trạng thái hóa đơn (đã phát hành/số hóa đơn/link tải) gắn với `top_ups`/`subscription_orders`.

**Effort**: **M–L** — không phải chỉ là code; cần chọn nhà cung cấp, có hợp đồng dịch vụ hóa đơn điện tử, và một model/migration mới liên kết đơn hàng ↔ hóa đơn. Việc này phụ thuộc quyết định kinh doanh (operator có bán cho doanh nghiệp cần khấu trừ VAT không, hay chủ yếu bán cho cá nhân/dev không cần hóa đơn).

### 3. Tài liệu tiếng Việt

**Hiện trạng repo**: `controller/misc.go:77` trả `docs_link` mặc định là `https://docs.newapi.pro` (`setting/operation_setting/general_setting.go:27`) — tài liệu upstream, tiếng Anh/Trung. Thư mục `docs/` của fork có 12+ file, bao gồm `docs/payments-sepay.md` — tài liệu vận hành SePay **viết hoàn toàn bằng tiếng Trung giản thể** (đọc trực tiếp nội dung: "# SePay 支付运营手册", "覆盖 SePay 银行转账支付的配置..."). `docs/translation-glossary.md` có 2 bản dịch phụ (`.fr.md`, `.ru.md`) nhưng không có `.vi.md`.

**Đánh giá**: Đây là gap rõ ràng và rẻ nhất để xử lý trong toàn bộ danh sách — tài liệu vận hành SePay (tính năng lõi của fork VN) đang không có bản tiếng Việt, chỉ có bản tiếng Trung. Chi phí thấp vì nội dung kỹ thuật đã có sẵn, chỉ cần dịch + đặt `docs_link` trỏ đến vị trí lưu trữ tài liệu VN (self-host một trang docs đơn giản, hoặc thư mục `docs/` render qua GitHub).

**Effort**: **S** — dịch thuật thuần túy, không đổi code (trừ việc cân nhắc đổi giá trị mặc định `docs_link` nếu operator tự host docs).

### 4. Model và nhà cung cấp liên quan đến VN

**Hiện trạng repo**: `constant/channel.go` có 119 dòng khớp mẫu (~118 hằng số kênh theo ground truth), `relay/channel/` có 45 thư mục adapter (`openai/`, `ali/`, `baidu/`, `zhipu/`, `tencent/`, `volcengine/`, `deepseek/`, `moonshot/`, `xai/`, `gemini/`, `claude/`, v.v.). Không có mục nào cho FPT, Viettel, VNG/Zalo, hay tên model PhoGPT/Vistral/VinaLLaMA.

**Mô hình tiếng Việt**: PhoGPT (VinAI Research, 2023, mô hình mở gốc tiếng Việt, PhoGPT-4B/PhoGPT-4B-Chat, 102B token huấn luyện) và Vistral được coi là 2 LLM tiếng Việt mở tham chiếu nhiều nhất trong nghiên cứu NLP tiếng Việt hiện nay (nguồn: emergentmind.com/topics/phogpt, arxiv 2311.02945, truy cập 2026-09-05). **Không tìm được bằng chứng các mô hình này có API thương mại/hosted phổ biến** — chúng chủ yếu là model mở tự host (Hugging Face), không phải dịch vụ API vận hành sẵn, nên việc thêm "channel adapter" theo đúng nghĩa (như cho OpenAI/Anthropic) ít có giá trị trừ khi operator tự host chúng qua vLLM/Ollama (đã có adapter `ollama/`, `xinference/` dùng chung được).

**Nhà cung cấp SEA có API OpenAI-compatible**:
- **GreenNode** (VNG Cloud sáp nhập với đơn vị hạ tầng AI, thương hiệu GreenNode) cung cấp MaaS với hơn 20 model mở/độc quyền, có endpoint OpenAI-compatible công khai: `https://maas-llm-aiplatform-hcm.api.vngcloud.vn/v1`, đặt tại trung tâm dữ liệu TP.HCM và Hà Nội (nguồn: GreenNode docs tiếng Việt "Kết nối OpenAI-compatible với GreenNode MaaS", truy cập 2026-09-05, https://docs.vngcloud.vn/vng-cloud-document/vn/ai-stack/agent-base/ai-coding/ket-noi-openai-compatible-voi-maas; DatacenterDynamics xác nhận việc sáp nhập thương hiệu, truy cập 2026-09-05).
- **FPT** đã trở thành OpenAI Select Partner tại VN (tích hợp OpenAI vào hệ sinh thái FPT, không phải nhà cung cấp model riêng có API cạnh tranh) — **chưa xác minh** được endpoint API "FPT AI Cloud" độc lập kiểu OpenAI-compatible trong phiên nghiên cứu này.
- **Viettel AI**: đang fine-tune Nvidia Nemotron 3 Super cho ứng dụng pháp lý quy mô quốc gia, định vị là nhà cung cấp hạ tầng AI chủ quyền — **chưa xác minh** có API công khai dạng OpenAI-compatible cho bên thứ ba hay không (nguồn: TechNode Global, truy cập 2026-09-05, https://technode.global/2026/06/03/vietnams-tech-giam-fpt-viettel-join-nvidias-sovereign-ai-push/).

**Model quốc tế được dev VN ưa chuộng**: không tìm được khảo sát/số liệu định lượng riêng cho VN trong phiên tìm kiếm này — **chưa xác minh**, không nên đưa ra khẳng định suy đoán.

**Kết luận**: nếu muốn phủ nhà cung cấp AI khu vực, **GreenNode là ứng viên rõ ràng và effort thấp nhất** vì đã OpenAI-compatible — có thể dùng ngay qua kênh "custom/openai-compatible" hiện có của new-api (không cần code adapter riêng), chỉ cần thêm vào danh sách preset nhà cung cấp trong UI nếu muốn trải nghiệm mượt hơn.

**Effort**: **S** (dùng qua custom base URL, không cần code) đến **S–M** (nếu muốn thêm preset có tên/logo trong UI chọn kênh).

### 5. Zalo (OAuth đăng nhập + ZNS thông báo)

**OAuth đăng nhập**: `oauth/registry.go` định nghĩa `Register(name string, provider Provider)` — hệ thống registry cho phép thêm provider mới không cần sửa core. `oauth/github.go` (183 dòng) là mẫu tham chiếu tốt để viết `oauth/zalo.go`. Zalo OAuth v3 dùng luồng chuẩn `authorize` → `code` → đổi `access_token` (hết hạn 3600s) qua `https://oauth.zaloapp.com/v3/auth` (nguồn: developers.zalo.me/docs/official-account/bat-dau/xac-thuc-va-uy-quyen-cho-ung-dung-new, truy cập 2026-09-05). Cần đăng ký App ID/Secret trên Zalo for Developers, và tên/mô tả app phải 20–500 ký tự theo yêu cầu duyệt app (nguồn: cùng trang). **Chưa xác minh rõ** mức độ nghiêm ngặt của quy trình duyệt app Zalo cho ứng dụng đăng nhập bên thứ 3 (một số API Zalo yêu cầu app đã được duyệt production mới cấp quyền lấy email/số điện thoại).

**ZNS (thông báo)**: `service/user_notify.go` có `switch notifyType` với 4 case: `Email`, `Webhook`, `Bark`, `Gotify` (dòng 66–107). Thêm case `Zalo` theo đúng mẫu này là thay đổi cục bộ, không phá vỡ contract. Zalo Notification Service (ZNS) API cho phép gửi tin nhắn theo template đã duyệt tới người dùng Zalo (không cần kết bạn OA trước, khác với tin nhắn CSKH thông thường) — nguồn chính thức: developers.zalo.me/docs/zalo-notification-service/bat-dau/gioi-thieu-zalo-notification-service-api, truy cập 2026-09-05. Giá tính theo loại template/số nút bấm, thu phí trên tin gửi thành công — **không tìm được bảng giá cụ thể bằng số trong phiên tìm kiếm này, chưa xác minh** (một trang quảng cáo "Bảng Giá ZNS 2026" xuất hiện nhưng nội dung số liệu không lấy được qua kết quả tìm kiếm).

**Kết luận**: cả 2 đều khả thi kỹ thuật theo pattern có sẵn trong repo. Giá trị thực tế phụ thuộc đối tượng người dùng — nếu là dev/API consumer thì Google/GitHub OAuth hiện có đã đủ; ZNS có giá trị hơn cho thông báo giao dịch (nạp tiền thành công, hết hạn gói) tới người dùng phổ thông không quen kiểm tra email.

**Effort**: OAuth **M** (code theo mẫu + quy trình duyệt app Zalo, có thể mất thời gian chờ ở phía Zalo, không phải effort code); ZNS **M** (thêm notify channel + đăng ký OA + tích hợp thanh toán dịch vụ ZNS).

### 6. Hiển thị giá bằng VND

**Hiện trạng repo — đã hỗ trợ qua cấu hình, không cần code**:
- Backend: `setting/operation_setting/general_setting.go` định nghĩa `QuotaDisplayTypeCustom = "CUSTOM"` cùng `CustomCurrencySymbol` (mặc định `"¤"`) và `CustomCurrencyExchangeRate` (mặc định `1.0`). `controller/misc.go` expose các field này (`custom_currency_symbol`, `custom_currency_exchange_rate`, `display_in_currency`, `usd_exchange_rate`, `price`) qua API status.
- Frontend: `web/src/lib/currency.ts` (case `'CUSTOM'` ở dòng 196 và 556) đọc đúng `customCurrencySymbol`/`customCurrencyExchangeRate` từ `system-config-store.ts` (kiểu `CurrencyDisplayType = 'USD' | 'CNY' | 'TOKENS' | 'CUSTOM'`). `web/src/features/pricing/lib/price.ts` → `formatPrice()` gọi `formatCurrencyFromUSD()` dùng chung store này. `web/src/features/home/components/sections/pricing-preview.tsx` dùng `usePricingData()` lấy `priceRate`/`usdExchangeRate` từ `status`, và `formatPrice` từ cùng thư viện — **cùng một cơ chế hiển thị tiền tệ với trang pricing đầy đủ**, không có code riêng lệch pha.

**Kết luận**: chỉ cần admin vào cài đặt chung, đặt `quota_display_type = CUSTOM`, `custom_currency_symbol = "₫"`, `custom_currency_exchange_rate = <tỷ giá VND/USD mong muốn>` — cả trang pricing và phần preview trên trang chủ sẽ hiển thị giá bằng VND ngay lập tức, không cần sửa code. Đây là gap giả — nên loại khỏi backlog kỹ thuật và chỉ cần một dòng hướng dẫn vận hành.

**Lưu ý vận hành liên quan (không phải gap, nhưng đáng nêu)**: `price` mặc định của SePay trong `design.md` (D5) là **1000 Dong/USD**, trong khi tỷ giá thị trường thực tế tại 2026-09-04 là khoảng **26.000–26.900 VND/USD** (tỷ giá trung tâm NHNN 25.605, tỷ giá thị trường ~26.072–26.269 VND/USD trong 30 ngày qua — nguồn: tổng hợp Vietcombank/Investing.com qua tìm kiếm, truy cập 2026-09-05). Giá trị `1000` trong code chỉ là **giá bán lẻ** (Dong operator thu cho mỗi USD credit — có thể có biên lợi nhuận khác tỷ giá thị trường), không phải lỗi; nhưng nếu operator quên cấu hình con số thực tế trước khi vận hành, người dùng có thể trả sai giá. Đây là điểm cần đưa vào checklist go-live, không phải thay đổi code.

### 7. Vùng/độ trễ

**Hiện trạng repo**: `docker-compose.yml:35` và `docker-compose.dev.yml:32` đặt `TZ=Asia/Shanghai` làm mặc định cho container ứng dụng. Không tìm thấy endpoint provider nào bị hard-code theo vùng cụ thể (các URL API nhà cung cấp AI là cấu hình qua channel settings, không hard-code trong compose).

**Bằng chứng thị trường**: Singapore là hub khu vực phổ biến, độ trễ tới Việt Nam khoảng 10–40ms qua cáp quang biển, đủ thấp cho API serving (nguồn: MassiveGRID blog "Low-Latency VPS for Asia-Pacific", truy cập 2026-09-05). VPS đặt tại Việt Nam cũng là lựa chọn hợp lệ nhưng không bắt buộc.

**Kết luận**: mục này nhỏ như ground truth đã nêu. `TZ=Asia/Shanghai` chỉ ảnh hưởng nhãn thời gian trong log/cron container, không ảnh hưởng độ trễ mạng thực tế — sửa thành `Asia/Ho_Chi_Minh` là thay đổi cosmetic 1 dòng trong file compose mẫu.

**Effort**: **S**.

### 8. Các vấn đề khác gặp trong tháng đầu vận hành

- **Xác thực số điện thoại**: `model/user.go` không có trường `Phone`; hệ thống chỉ có `EmailVerificationEnabled` (`controller/misc.go:56`, `controller/user.go:261`). Không có bằng chứng cụ thể đây là rào cản — nhiều dịch vụ API-first ở VN (kể cả các đối thủ) vẫn chỉ dùng email. **Không đề xuất làm ngay**, cần hỏi lại nhu cầu thực tế (ví dụ: có cần theo Nghị định về định danh người dùng dịch vụ số không) trước khi lên kế hoạch.
- **Thuế trên nạp tiền/thu nhập của operator**: không có logic thuế trong `controller/topup.go`/`controller/sepay.go` (đúng như kỳ vọng — đây không phải trách nhiệm của phần mềm mà là nghĩa vụ kế toán của operator). Không phải gap kỹ thuật.
- **`custom_currency_exchange_rate` vs `price` (SePay) là 2 con số riêng biệt** — dễ nhầm lẫn khi vận hành nếu admin chỉ cập nhật 1 trong 2. Nên ghi rõ trong tài liệu tiếng Việt (mục 3) rằng đây là 2 cấu hình độc lập: `price` (trong SePay settings) quyết định số Dong người dùng phải trả khi nạp, còn `custom_currency_exchange_rate` (trong general settings) chỉ quyết định cách hiển thị số dư/giá — chúng nên được đặt cùng giá trị để tránh hiển thị sai lệch với số tiền thực trả, nhưng hệ thống không tự đồng bộ 2 giá trị này.

---

## (d) Khuyến nghị 2–3 việc làm trước

1. **Dịch `docs/payments-sepay.md` (và các phần vận hành liên quan) sang tiếng Việt, thêm `docs/translation-glossary.vi.md`.** Đây là tính năng lõi của chính fork này (SePay = thanh toán VN) nhưng tài liệu vận hành hiện đang bằng tiếng Trung — nghịch lý rõ nhất tìm được trong repo. Effort thấp nhất (S), giá trị vận hành cao nhất, không đụng code sản phẩm nên rủi ro gần bằng không.

2. **Viết hướng dẫn admin 1 trang: cách bật hiển thị giá VND** (`quota_display_type=CUSTOM`, `custom_currency_symbol=₫`, `custom_currency_exchange_rate=<tỷ giá>`) và lưu ý đồng bộ với `price` trong SePay settings. Đây không phải việc code — là việc viết tài liệu — nhưng cần làm trước vì nếu không, đội ngũ có thể lãng phí effort đi code lại thứ đã tồn tại (rủi ro thực tế đã thấy trong yêu cầu nghiên cứu này, vốn đặt câu hỏi "cần code hay admin config đã đủ" — nay có câu trả lời dứt khoát).

3. **Đưa ra quyết định kinh doanh về hóa đơn điện tử trước khi lên kế hoạch kỹ thuật** — đây là gap thật duy nhất có effort lớn (M–L). Cần hỏi chủ dự án: đối tượng khách hàng có bao gồm doanh nghiệp cần khấu trừ VAT không? Nếu có, nên chọn nhà cung cấp (MISA meInvoice có Open API công khai, dễ tiếp cận nhất trong 3 nguồn tìm được) và làm rõ phạm vi (phát hành theo yêu cầu, không phải bắt buộc mọi giao dịch) trước khi viết spec. Không nên tự ý triển khai nếu chưa xác nhận nhu cầu — đây là quyết định phạm vi kinh doanh, không phải quyết định kỹ thuật thuần túy.

*(Zalo OAuth/ZNS và adapter GreenNode/FPT xếp sau 3 việc trên vì effort tương đương hoặc cao hơn nhưng bằng chứng nhu cầu yếu hơn — nên làm sau khi có tín hiệu người dùng thực tế yêu cầu, hoặc gộp vào một đợt lên kế hoạch riêng nếu operator xác nhận ưu tiên.)*

---

## (e) Câu hỏi chưa trả lời được

1. Nghị định 254/2026/NĐ-CP (hiệu lực 01/07/2026, thay thế Nghị định 123/2020) — chưa đọc được toàn văn, chỉ có mô tả gián tiếp qua các trang tổng hợp luật. Cần tra cứu Cổng thông tin Chính phủ hoặc Thư viện Pháp luật để xác nhận nội dung chi tiết trước khi thiết kế tính năng hóa đơn.
2. Viettel S-Invoice và VNPT Invoice có Open API công khai dạng tài liệu lập trình viên (như MISA) hay không, và điều kiện hợp tác/giá — chưa xác minh, cần liên hệ trực tiếp hoặc tìm developer portal riêng.
3. FPT AI Cloud và Viettel AI có endpoint API công khai dạng OpenAI-compatible cho bên thứ ba (ngoài các dự án AI chủ quyền cấp quốc gia) hay không — chưa xác minh được trong phiên nghiên cứu này.
4. Bảng giá cụ thể của Zalo ZNS (VND/tin nhắn theo loại template) — không lấy được số liệu cụ thể qua tìm kiếm, cần tra cứu trực tiếp `developers.zalo.me` hoặc liên hệ Zalo/đại lý.
5. Mức độ nghiêm ngặt của quy trình Zalo duyệt app cho luồng OAuth đăng nhập bên thứ 3 (đặc biệt quyền lấy số điện thoại/email người dùng) — chưa xác minh, có thể ảnh hưởng effort thực tế của gap #4/#5 trong bảng ưu tiên.
6. Khảo sát định lượng về model AI quốc tế mà lập trình viên Việt Nam ưa chuộng nhất (GPT, Claude, Gemini, DeepSeek, Qwen...) — không tìm được số liệu riêng cho VN, không nên suy đoán.
7. SePay có tính phí giao dịch/gói dịch vụ theo số lượng giao dịch hàng tháng (thấy nhắc "chọn gói phù hợp" khi đăng ký) — chưa lấy được bảng giá cụ thể, nên xác minh trước khi so sánh chi phí với MoMo/VNPay nếu sau này cân nhắc đa dạng hóa cổng thanh toán.
8. Có quy định pháp lý nào của Việt Nam bắt buộc xác thực số điện thoại cho dịch vụ số/thanh toán trực tuyến áp dụng cho mô hình kinh doanh này hay không — chưa xác minh, cần tư vấn pháp lý riêng, không suy đoán từ tìm kiếm chung.

---

```
Status: DONE_WITH_CONCERNS
Summary: Phân tích khoảng trống hoàn tất với bằng chứng repo + nguồn web có trích dẫn; phát hiện quan trọng nhất là hiển thị giá VND đã khả thi qua cấu hình (không cần code) và tài liệu vận hành SePay hiện bằng tiếng Trung thay vì tiếng Việt. Gap thật lớn nhất (hóa đơn điện tử) cần quyết định kinh doanh trước khi lên kế hoạch kỹ thuật.
Report: D:/Dev/www/api-gateway/plans/reports/research-260905-0020-vn-market-feature-gaps.md
Concerns/Blockers: Một số nguồn web quan trọng (toàn văn Nghị định 254/2026, API Viettel/VNPT Invoice, bảng giá Zalo ZNS, endpoint công khai FPT AI Cloud/Viettel AI) không xác minh được đầy đủ trong phiên nghiên cứu này — liệt kê ở mục (e), nên tra cứu bổ sung trước khi cam kết phạm vi cho các gap #2, #4, #5.
```
