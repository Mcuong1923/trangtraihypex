package main

import "strings"

// Thư viện lời thoại NPC (PLAN_LOGIC_GAME 5.3). Giọng hài, trêu nhẹ chuyện trong game; không bàn ngoại hình,
// giới tính hay đời thực. {ten}: tên gà người chơi; {anh}: "anh"/"chị"/"bạn" theo giới tính hồ sơ;
// {giun}, {cap}, {khu}, {eta}, {n}: điền theo ngữ cảnh. Máy chủ chọn câu; client chỉ hiện.

var NPCLines = map[string][]string{
	// --- Cụ Cục & hướng dẫn người mới ---
	"elder.welcome_new": {"Ơ kìa, gà mới! {ten} đấy hả? Chào mừng tới Trại Hypex — nơi giun béo, thóc đầy, và chưa ai bị làm gà rán. Lại đây cụ dặn mấy câu!"},
	"elder.welcome_back": {
		"Về rồi đấy à {ten}! Làng nhớ cháu lắm… nhất là Lợn Ủn, nó nhớ giun của cháu.",
		"{ten} đó hả? Hôm nay trời đẹp, giun chắc ngoi lên nhiều lắm đấy.",
	},
	"elder.checkin":  {"Điểm danh ngày {n}! Cụ gửi {ten} chút quà, tiêu cho khéo nhé."},
	"elder.levelup":  {"Lên cấp {cap} rồi à {ten}? Giỏi! Quà cụ đã gửi vào túi rồi đấy.", "Cấp {cap}! Cứ đà này mai mốt làm trưởng thôn luôn."},
	"elder.hungry":   {"Bụng cháu réo to quá, cụ ngồi đây còn nghe! Có giun thì ăn sống một con đi."},
	"elder.tut_done": {"Giỏi lắm {ten}! Đào được, bán được, ăn được — vậy là thành gà làng chính hiệu. Cụ gửi thưởng qua Chim Chíp rồi nhé!"},
	"guide.run":      {"Thử chạy coi! Giữ Shift hoặc nút 🏃. Chạy lâu hụt hơi thì nghỉ 5 giây — gà chứ có phải siêu nhân đâu."},
	"bird.tut_bus":   {"Lên xe đi bà con! Lượt đầu miễn phí, lượt sau cũng… miễn phí, tới lượt thứ ba mới tính tiền!"},
	"pig.tut_dig":    {"Lần đầu đào hả? Mổ nhẹ thôi kẻo giun nó kiện! Giữ nút để kéo, thấy đỏ thì thả ra."},
	"pig.tut_sell":   {"Giun đất 5 thóc, giun vàng 150 thóc. Giun cầu vồng thì… cháu đào được, ta mua cả họ."},
	"food.tut_eat":   {"Ăn đi cho có sức. Cám thì rẻ, suất thì no, ăn sống thì… cũng được, gà mà!"},

	// --- Chủ quán, NPC cố định ---
	"pig.greet":      {"Giun về chợ đây! Hôm nay {giun} được giá, tôi trả thêm 10% đó nha.", "Có giun không bán cho tôi, tôi trả giá đẹp hơn mặt tôi!"},
	"pig.rare":       {"Ui chà, {giun}! Bán cho tôi đi, hôm nay giá đẹp!"},
	"pig.garden":     {"Đào giun ở đây nè! Giun hiếm quẫy mạnh lắm, giữ chắc mỏ vào."},
	"goat.coupon":    {"Ồ khách mới! Hôm nay giảm 10% cho xe đầu tiên, hạn 24 giờ. Đi bộ hoài teo cả đùi gà đó!"},
	"goat.can_bike":  {"Đủ tiền mua xe đạp rồi đó, đạp cho thon đùi!"},
	"goat.greet":     {"Xe nào cũng có, chỉ thiếu mỗi xe bay. Ghé coi đi!", "Hôm nay tiệm mới thay nhớt, xe chạy êm như ru."},
	"duck.eggs":      {"Trứng để trong túi lâu là thành trứng luộc đó! Đem đây cô ấp cho.", "Ổ ấp đang trống, cháu có trứng thì đưa cô nhé."},
	"duck.greet":     {"Trạm ấp hôm nay ấm áp lắm. Nở ra con gì thì… tuỳ duyên!"},
	"cow.land":       {"Muốn có nhà riêng không? Gặp ông Chủ Đầu Tư bên kia, rồi về đây tôi xây cho!"},
	"cow.greet":      {"Rơm thì nhanh, gỗ thì chắc, gạch thì sang — còn thóc thì… phải có!"},
	"investor.greet": {"Đất ven ao, view triệu đô, sổ đỏ… à sổ rơm chính chủ! Mua ngay kẻo hết lô đẹp!"},
	"investor.broke": {"Chưa đủ thóc à? Không sao, đào thêm vài con giun vàng là có ngay!"},
	"bird.stop":      {"Xe tới sau {eta} giây! Còn {n} lượt miễn phí đó."},
	"dog.greet":      {"Gâu! Muốn lên sàn thử sức không? Luật đơn giản: mổ nhanh hơn đối thủ là thắng."},
	"food.greet":     {"Cám mới xay, rau mới hái! Ăn no rồi hẵng đào tiếp nha."},
	"fashion.greet":  {"Áo mới về, mặc vào là cả làng ngoái nhìn!"},
	"fashion.crown":  {"Ồ, vua giun về làng!"},
	"frog.greet":     {"Ộp! Bùn ven suối nhiều giun ngọc lắm đó."},

	// --- Xe buýt ---
	"bus.arrive": {"Bến {khu} đây bà con!", "Tới {khu} rồi, xuống nhớ cầm đồ nha!"},
	"bus.depart": {"Xe chạy đây, bám chắc nha!", "Qua ổ gà đó, ngồi yên nghen!"},

	// --- Xe ôm ---
	"xeom.offer": {
		"Ê, đi đâu đó? Lên xe anh chở, 1 thóc thôi, nhanh gấp đôi chạy bộ!",
		"Trời nắng vậy đi bộ chi cho mệt, lên đây anh chở, máy lạnh tự nhiên!",
		"Xe ôm đây! Chở tận cổng, khỏi lạc đường!",
	},
	"xeom.decline": {"Thôi được, đi bộ cho khoẻ, gặp lại ở ngã ba nha!", "Không đi thì thôi, anh đi kiếm khách khác, khỏi năn nỉ!"},
	"xeom.ride":    {"Bám chắc nha, qua ổ gà đó!", "Hồi xưa anh cũng đào giun, giờ chạy xe ôm thu nhập ổn định hơn!", "Đường này anh thuộc như lòng bàn cánh!"},
	"xeom.thanks":  {"Tới rồi! Nhớ đánh giá 5 sao… à ở đây chưa có sao, cảm ơn là được!", "Tới nơi an toàn! Lần sau gọi anh nha."},

	// --- Giao thông NPC ---
	"traffic.honk":  {"Bíp bíp! Tránh đường nha!", "Cho qua với, cảm ơn nha!"},
	"traffic.hello": {"Chào buổi sáng!", "Hôm nay giun béo không?", "Đi đâu mà vội thế?"},

	// --- Cameo ---
	"rich.mock":       {"Ơ kìa, thời buổi này còn có gà đi bộ à?", "Ủa, đi bộ hả? Tưởng tập thể dục chứ!"},
	"rich.laugh":      {"Hahaha! Bíp bíp! Mua xe đi cưng!", "Bye bye, ăn khói nhé! Hahaha!"},
	"rich.bike":       {"Xe đạp à? Dễ thương ghê, như đồ chơi!"},
	"rich.respect":    {"Ồ, xe xịn đấy! Hôm nào đua một vòng không?"},
	"mom.lesson":      {"Con thấy chưa, không chịu học thì sau này lại như {anh} kia đấy."},
	"chick.reply":     {"Nhưng {anh} ấy dễ thương mà mẹ…", "Con sẽ học đào giun thật giỏi!", "Mẹ ơi, {anh} ấy đang nhìn mình kìa!"},
	"shipper.ask":     {"Ơ cho hỏi {khu} đi lối nào vậy?"},
	"shipper.thanks":  {"Cảm ơn nha! Đây, biếu 1 thóc uống nước."},
	"shipper.lost":    {"Chỉ đường kiểu gì vậy trời! Lạc mất một vòng!"},
	"reporter.ask":    {"Xin chào! Cảm nghĩ khi đào được {giun}?"},
	"reporter.thanks": {"Tuyệt vời! Lên trang nhất báo Làng Hypex sáng mai!"},
	"vendor.offer":    {"Cám nóng đây! Đắt hơn ngoài chợ tí xíu nhưng giao tận mỏ!"},
	"vendor.thanks":   {"Ngon thì nhớ ủng hộ lần sau nha!"},
	"elder.exercise":  {"Chạy gì mà như ma đuổi! Hít vào… thở ra… Từ từ thôi con."},

	// --- Phản ứng theo trạng thái ---
	"react.hungry":  {"Bụng kêu to thế, ra chợ ăn gì đi!", "Không có thóc thì ăn sống một con giun đi, gà mà!"},
	"react.escape":  {"Nó chuồn rồi! Lần sau đừng giật mạnh thế.", "Giun trơn như lươn, tuột mất tiêu!"},
	"react.rare":    {"Trời ơi, {giun} kìa! Đỏ mắt luôn!", "Ê ê, {giun} kìa bà con ơi!"},
	"react.levelup": {"Lên cấp rồi kìa! Chúc mừng nha!", "Cấp {cap}! Đãi cả làng đi chứ!"},
	"react.lostpk":  {"Thua keo này ta bày keo khác!", "Không sao, mai tập thêm là thắng!"},

	// --- Trận biểu diễn ở Sân tỷ thí ---
	"ref.start":   {"Hai bên chào nhau! Không mổ mắt, không đá dưới cánh! 3… 2… 1… Bắt đầu!"},
	"ref.end":     {"KO! {f} thắng! Cả hai về ăn cám rồi tập tiếp nhé."},
	"ref.invite":  {"Muốn lên sàn thử không? Gặp tôi nhé!"},
	"crowd.cheer": {"Cố lên {f}!", "Mổ đi, mổ đi!", "Hay quá, làm lại phát nữa!"},
	"crowd.bet":   {"Tôi cược 5 thóc {f} thắng!", "Kèo này {f} ăn chắc!"},
	"crowd.mock":  {"Mổ đi chứ, gãi ngứa à!", "Đánh như múa vậy trời!", "Ơ kìa, ngã rồi, sao vàng bay quanh đầu kìa!"},
	"crowd.end":   {"Thôi về, mai xem tiếp!", "Trận hay thật!"},
}

// pronoun chọn ngôi thứ ba theo giới tính hồ sơ (khác/ẩn dùng "bạn").
func pronoun(a *Account) string {
	if a == nil || a.Char == nil {
		return "bạn"
	}
	switch a.Char.Sex {
	case "nam":
		return "anh"
	case "nu":
		return "chị"
	}
	return "bạn"
}

// fillLine điền các chỗ trống trong câu thoại.
func fillLine(s string, kv map[string]string) string {
	for k, v := range kv {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}
