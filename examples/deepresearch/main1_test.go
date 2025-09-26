package main

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestMain1(t *testing.T) {
	messages := []*schema.Message{
		schema.UserMessage("马尔代夫的珊瑚为什么白化了？"),
		schema.AssistantMessage("", []schema.ToolCall{
			{
				ID:   "1",
				Type: "function",
				Function: schema.FunctionCall{
					Name:      "google_search",
					Arguments: `{"query": "马尔代夫 珊瑚 白化 原因 2025"}`,
				},
			},
		}),
		schema.ToolMessage(
			"{\"query\":\"马尔代夫 珊瑚 白化 原因 2025\",\"results\":[{\"title\":\"監測珊瑚礁復原力\",\"url\":\"https://reefresilience.org/zh-TW/topic/monitoring-reef-resilience/\",\"description\":\"by 米歇爾·格勞蒂 | 2025 年5 月16 日. 大規模珊瑚白化已導致大面積珊瑚礁消失，凸顯了建立全球協調監測系統的必要性。本評論分析了60 年來的白化數據，這些數據來自三個全球資料庫（1963 年至2022 年）以及對珊瑚礁管理者和科學家的調查。研究結果凸顯了監測工作在標準化、地理覆蓋範圍和數據一致性方面存在重大差距，這些因素限制了我們了解白化 ...\"},{\"title\":\"珊瑚白化|珊瑚礁復原力網絡\",\"url\":\"https://reefresilience.org/zh-TW/category/module/coral-bleaching/\",\"description\":\"2025 年4 月24 日 | 珊瑚漂白, 模塊, 威脅. 雖然當地管理部門無法直接控制大規模珊瑚白化的原因，但珊瑚礁管理者在白化事件之前、期間和之後發揮重要作用。他們的職責通常包括預測和傳達風險、評估影響、了解對珊瑚礁復原力的影響以及實施管理行動以減輕損害的嚴重程度並支持珊瑚礁恢復。白化因應計畫描述了檢測、評估和應對白化事件的步驟。\"},{\"title\":\"馬爾地夫的珊瑚嚴重白化\",\"url\":\"https://www.worldpeoplenews.com/content/news/7871\",\"description\":\"國際自然保護聯合(IUCN)警告，全球暖化，海水溫度上升，擁有全世界3%珊瑚礁的印度洋島國馬爾地夫，珊瑚礁出現嚴重白化現象。 根據IUCN的聲明，馬爾地夫的珊瑚已有60%白化，在一部分海域更高達90%。研究是由馬爾地夫海洋研究中心(MRC)和美國環境保護局(EPA)共同進行調查提供的數據。\"},{\"title\":\"大規模白化與海星爆發澳洲大堡礁珊瑚覆蓋率創39年最大損失\",\"url\":\"https://www.natgeomedia.com/environment/article/content-18411.html\",\"description\":\"Sep 1, 2025 — 澳洲珊瑚長期監測報告指出，受全球大規模珊瑚白化、風暴、棘冠海星等因素影響，大堡礁的硬珊瑚覆蓋率大幅下降，三個區中有兩個創下39年來最嚴重下滑紀錄。- 國家地理雜誌中文網.\"},{\"title\":\"「這太可怕」全球84%珊瑚礁遇最嚴重白化為環境禱告並從 ...\",\"url\":\"https://cdn-news.org/news/N2504240003\",\"description\":\"Apr 24, 2025 — 大面積珊瑚白化現象首次出現在1980年代的加勒比海地區，原因即「海水溫度上升」。自1998年來，世界上最大的珊瑚礁系統—澳洲的大堡礁，已經歷七次大規模白化事件，其中五次 ...\"},{\"title\":\"全球逾八成珊瑚白化\",\"url\":\"http://www.news.cn/asia/20250504/4827c86520d1440d9fbe61ecab4955a6/c.html\",\"description\":\"Apr 24, 2025 — 这次大规模白化始于2023年，影响波及太平洋、印度洋和大西洋地区。 负责跟踪全球珊瑚状况的美国国家海洋和大气管理局说：“自2023年1月1日至2025年4月20日，引发珊瑚白化的高温已影响83.7%的珊瑚生长区……当前没有放缓趋势。” “国际珊瑚礁倡议”组织负责新闻事务的马克·埃金说，珊瑚白化现象“正在彻底改变地球面貌以及 ...\"},{\"title\":\"气候变化对珊瑚礁的影响及2024-2025年最新研究与应对策略\",\"url\":\"https://www.forwardpathway.com/115626\",\"description\":\"Aug 6, 2024 — 本文深入探讨了气候变化对全球珊瑚礁的严峻威胁，特别是强调了海洋变暖、酸化及极端天气事件对珊瑚白化和生存的影响。文章引用了多项最新研究，包括加州大学圣地亚哥分校利用三维模型追踪夏威夷毛伊岛珊瑚白化反应的研究，以及关于珊瑚热耐受性和适应潜力的发现。研究表明，一些珊瑚种类展现出适应性，为未来的恢复项目提供了希望。\"},{\"title\":\"珊瑚白化擴及全球84%礁區規模創紀錄| 海洋熱浪| 珊瑚礁\",\"url\":\"https://www.epochtimes.com/b5/25/4/23/n14489892.htm\",\"description\":\"Apr 23, 2025 — 根據最新報告，已有約84%的珊瑚礁區域出現白化現象，範圍橫跨印度洋、大西洋與太平洋。 專家認為，此現象與持續升高的海水溫度有關，導致大量珊瑚失去色彩甚至死亡。當海水溫度異常升高時 ...\"},{\"title\":\"大堡礁珊瑚创纪录大幅下降\",\"url\":\"https://www.dw.com/zh/%E5%A4%A7%E5%A0%A1%E7%A4%81%E7%8F%8A%E7%91%9A%E5%88%9B%E7%BA%AA%E5%BD%95%E5%A4%A7%E5%B9%85%E4%B8%8B%E9%99%8D/a-73556549\",\"description\":\"Aug 10, 2025 — 报告称，大堡礁经历了前所未有的热浪，导致迄今范围最广、最严重的珊瑚白化事件。 调查发现，自1986年开始监测以来，大堡礁三个地区中有两个发生珊瑚数量最大幅度的下降。 报告 ...\"}]}",
			"1"),
	}
	compressGoogleSearchResponse(context.Background(), messages, 2, messages[2])
}
