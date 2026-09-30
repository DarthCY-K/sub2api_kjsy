package service

import (
	"fmt"
	"strings"
)

// 排行榜的供应商归类：只用于展示筛选，按模型名关键字宽松匹配（如 opus → claude、k3 → moonshot）。
// 与路由用的 DetectModelPlatform（刻意不猜测、失败即拒绝）是两套规则，不要互相替代。
const (
	LeaderboardVendorClaude   = "claude"
	LeaderboardVendorOpenAI   = "openai"
	LeaderboardVendorGemini   = "gemini"
	LeaderboardVendorDeepSeek = "deepseek"
	LeaderboardVendorMoonshot = "moonshot"
	LeaderboardVendorZhipu    = "zhipu"
	LeaderboardVendorQwen     = "qwen"
	LeaderboardVendorMiniMax  = "minimax"
	LeaderboardVendorXAI      = "xai"
	LeaderboardVendorDoubao   = "doubao"
	LeaderboardVendorMistral  = "mistral"
	LeaderboardVendorMeta     = "meta"
	LeaderboardVendorOther    = "other"
)

var leaderboardVendorRules = []struct {
	vendor   string
	keywords []string
}{
	// 顺序即优先级：品牌关键字明确的在前，OpenAI 的短前缀（o1/gpt）放最后避免误吞。
	{LeaderboardVendorClaude, []string{"claude", "anthropic", "opus", "sonnet", "haiku"}},
	{LeaderboardVendorDeepSeek, []string{"deepseek"}},
	{LeaderboardVendorMoonshot, []string{"kimi", "moonshot"}},
	{LeaderboardVendorZhipu, []string{"glm", "zhipu", "bigmodel", "cogview", "cogvideo", "z-ai/"}},
	{LeaderboardVendorQwen, []string{"qwen", "qwq", "qvq", "tongyi"}},
	{LeaderboardVendorMiniMax, []string{"minimax", "abab"}},
	{LeaderboardVendorXAI, []string{"grok", "x-ai/", "xai/"}},
	{LeaderboardVendorDoubao, []string{"doubao", "seedream", "seedance"}},
	{LeaderboardVendorMistral, []string{"mistral", "mixtral", "codestral", "pixtral", "magistral", "devstral", "ministral", "voxtral"}},
	{LeaderboardVendorMeta, []string{"llama"}},
	{LeaderboardVendorGemini, []string{"gemini", "gemma", "learnlm", "imagen", "veo-", "nano-banana"}},
	{LeaderboardVendorOpenAI, []string{"openai", "gpt", "codex", "dall-e", "whisper", "sora", "davinci", "babbage", "text-embedding", "text-moderation", "omni-moderation"}},
}

var leaderboardVendorSet = func() map[string]struct{} {
	m := map[string]struct{}{LeaderboardVendorOther: {}}
	for _, r := range leaderboardVendorRules {
		m[r.vendor] = struct{}{}
	}
	return m
}()

// LeaderboardVendorOf 把模型名归到供应商；无法识别归 other。
func LeaderboardVendorOf(model string) string {
	s := strings.ToLower(strings.TrimSpace(model))
	base := s
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		base = s[i+1:]
	}
	if lbHasSeriesPrefix(base, 'k') { // Kimi 简写：k2 / k3-256k / k2.5
		return LeaderboardVendorMoonshot
	}
	for _, r := range leaderboardVendorRules {
		for _, kw := range r.keywords {
			if strings.Contains(s, kw) {
				return r.vendor
			}
		}
	}
	if lbHasSeriesPrefix(base, 'o') || strings.HasPrefix(base, "tts-") {
		return LeaderboardVendorOpenAI
	}
	return LeaderboardVendorOther
}

// ParseLeaderboardVendor 空串 = 不筛选。
func ParseLeaderboardVendor(vendor string) (string, error) {
	if vendor == "" {
		return "", nil
	}
	if _, ok := leaderboardVendorSet[vendor]; ok {
		return vendor, nil
	}
	return "", fmt.Errorf("invalid vendor %q", vendor)
}

// lbHasSeriesPrefix 判断 base 是否形如 <letter><数字...>，其后为结尾或 - . _ : 分隔符。
func lbHasSeriesPrefix(base string, letter byte) bool {
	if len(base) < 2 || base[0] != letter {
		return false
	}
	i := 1
	for i < len(base) && base[i] >= '0' && base[i] <= '9' {
		i++
	}
	if i == 1 {
		return false
	}
	return i == len(base) || strings.IndexByte("-._:", base[i]) >= 0
}
