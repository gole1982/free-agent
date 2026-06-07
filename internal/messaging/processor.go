package messaging

import (
	"regexp"
	"strings"
)

// ProcessorConfig 消息处理器配置
type ProcessorConfig struct {
	EnableAdFiltering     bool // 是否启用广告过滤
	EnableURLFiltering    bool // 是否启用URL过滤
	EnableXSSProtection   bool // 是否启用XSS防护
	EnableLengthTruncation bool // 是否启用长度截断
	MaxMessageLength      int  // 最大消息长度
}

// DefaultConfig 返回默认配置（安全模式）
func DefaultConfig() ProcessorConfig {
	return ProcessorConfig{
		EnableAdFiltering:     true,
		EnableURLFiltering:    true,
		EnableXSSProtection:   true,
		EnableLengthTruncation: true,
		MaxMessageLength:      10000,
	}
}

// PentestConfig 返回渗透测试配置（禁用安全限制）
func PentestConfig() ProcessorConfig {
	return ProcessorConfig{
		EnableAdFiltering:     true,  // 广告过滤仍然启用
		EnableURLFiltering:    true,  // URL过滤仍然启用
		EnableXSSProtection:   false, // 禁用XSS防护
		EnableLengthTruncation: false, // 禁用长度截断
		MaxMessageLength:      0,
	}
}

// MessageProcessor 统一消息处理器
type MessageProcessor struct {
	config           ProcessorConfig
	adKeywords       []string
	allowedURLPatterns []*regexp.Regexp
}

// NewMessageProcessor 创建消息处理器（使用默认配置）
func NewMessageProcessor() *MessageProcessor {
	return NewMessageProcessorWithConfig(DefaultConfig())
}

// NewMessageProcessorWithConfig 使用指定配置创建消息处理器
func NewMessageProcessorWithConfig(config ProcessorConfig) *MessageProcessor {
	return &MessageProcessor{
		config: config,
		adKeywords: []string{
			"ChatBYOK", "ChatGPT experience", "Telegram",
			"the best native", "experience in",
			"chatbyok.com", "chatbyok", "t.me/",
			"native ChatGPT", "best ChatGPT", "Try ChatGPT",
			"Powered by", "Visit us at", "Follow us on",
			"Subscribe to", "Join our", "Check out our",
			"Learn more at", "Get started at", "Sign up at",
			"Register at", "Download our", "Install our",
			"Use our", "Try our", "Buy our", "Purchase our",
			"Order our", "Get your", "Claim your", "Win a",
			"Free trial", "Limited time", "Special offer",
			"Discount code", "Promo code", "Coupon code",
			"Save up to", "Best price", "Lowest price",
			"Cheapest", "Most affordable",
		},
		allowedURLPatterns: []*regexp.Regexp{
			regexp.MustCompile(`docs\.[a-zA-Z]+\.[a-zA-Z]+`),
			regexp.MustCompile(`github\.com/[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+`),
			regexp.MustCompile(`readme`),
		},
	}
}

// CleanMessage 完整的消息清洗流程
func (p *MessageProcessor) CleanMessage(content string) string {
	if p.config.EnableAdFiltering {
		content = p.removeAds(content)
		content = p.removeSeparators(content)
	}
	if p.config.EnableURLFiltering {
		content = p.removeSpamURLs(content)
	}
	content = p.normalizeWhitespace(content)
	
	// 根据配置决定是否应用安全限制
	if p.config.EnableXSSProtection {
		content = p.SanitizeForHTML(content)
	}
	if p.config.EnableLengthTruncation && p.config.MaxMessageLength > 0 {
		content = p.TrimExcessiveLength(content, p.config.MaxMessageLength)
	}
	
	return strings.TrimSpace(content)
}

// removeAds 移除广告关键词和广告行
func (p *MessageProcessor) removeAds(content string) string {
	lines := strings.Split(content, "\n")
	filtered := []string{}
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		
		if p.isAdLine(trimmed) {
			continue
		}
		
		filtered = append(filtered, line)
	}
	
	content = strings.Join(filtered, "\n")
	
	for _, ad := range p.adKeywords {
		content = strings.ReplaceAll(content, ad, "")
		content = strings.ReplaceAll(content, strings.ToLower(ad), "")
		content = strings.ReplaceAll(content, strings.Title(ad), "")
	}
	
	return content
}

// isAdLine 判断是否为广告行
func (p *MessageProcessor) isAdLine(line string) bool {
	lowerLine := strings.ToLower(line)
	adCount := 0
	
	for _, ad := range p.adKeywords {
		if strings.Contains(lowerLine, strings.ToLower(ad)) {
			adCount++
		}
	}
	
	if adCount >= 2 {
		return true
	}
	
	spamPatterns := []string{
		"chatgpt experience",
		"native chatgpt",
		"telegram",
		"best chatgpt",
	}
	
	for _, pattern := range spamPatterns {
		if strings.Contains(lowerLine, pattern) {
			return true
		}
	}
	
	return false
}

// removeSeparators 移除分隔符行
func (p *MessageProcessor) removeSeparators(content string) string {
	lines := strings.Split(content, "\n")
	filtered := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		
		// 检查是否是分隔符行
		sepCount := 0
		for _, ch := range trimmed {
			if ch == '-' || ch == '=' || ch == '*' || ch == '_' || ch == '~' {
				sepCount++
			}
		}
		
		if sepCount <= len(trimmed)/2 || len(trimmed) <= 5 {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

// removeSpamURLs 移除广告URL
func (p *MessageProcessor) removeSpamURLs(content string) string {
	lines := strings.Split(content, "\n")
	filtered := []string{}
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 检查是否包含URL
		hasURL := strings.Contains(trimmed, "http://") || 
			strings.Contains(trimmed, "https://") || 
			strings.Contains(trimmed, "www.")
		
		if !hasURL {
			filtered = append(filtered, line)
			continue
		}
		
		// 检查是否是允许的URL
		isAllowed := false
		lowerLine := strings.ToLower(trimmed)
		for _, pattern := range p.allowedURLPatterns {
			if pattern.MatchString(lowerLine) {
				isAllowed = true
				break
			}
		}
		
		if isAllowed {
			filtered = append(filtered, line)
		}
	}
	
	return strings.Join(filtered, "\n")
}

// normalizeWhitespace 规范化空白字符
func (p *MessageProcessor) normalizeWhitespace(content string) string {
	// 移除多余的空行
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	
	// 移除行尾空格
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	
	return strings.Join(lines, "\n")
}

// ExtractCodeBlocks 从消息中提取代码块
func (p *MessageProcessor) ExtractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	
	re := regexp.MustCompile("```(\\w+)\\s*\\n([\\s\\S]*?)```")
	matches := re.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Code:     strings.TrimSpace(match[2]),
			})
		}
	}
	
	return blocks
}

// CodeBlock 代码块结构
type CodeBlock struct {
	Language string
	Code     string
}

// SanitizeForHTML 安全处理HTML内容（防止XSS）
func (p *MessageProcessor) SanitizeForHTML(content string) string {
	// 基本的XSS防护
	content = strings.ReplaceAll(content, "<script", "&lt;script")
	content = strings.ReplaceAll(content, "</script", "&lt;/script")
	content = strings.ReplaceAll(content, "<iframe", "&lt;iframe")
	content = strings.ReplaceAll(content, "javascript:", "javascript&#58;")
	return content
}

// TrimExcessiveLength 截断过长的消息
func (p *MessageProcessor) TrimExcessiveLength(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength-3] + "..."
}
