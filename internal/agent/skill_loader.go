package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SkillConfig 表示从 SKILL.md 文件的解析结果
type SkillConfig struct {
	Name        string
	Character   string
	Capabilities []string
	Metrics     *AgentTraits
	Content     string
}

// SkillLoader 负责加载和管理 Agent Skills
type SkillLoader struct {
	skillsDir string
	cache     map[string]*SkillConfig
}

func NewSkillLoader(skillsDir string) *SkillLoader {
	return &SkillLoader{
		skillsDir: skillsDir,
		cache:     make(map[string]*SkillConfig),
	}
}

// LoadSkill 从文件加载 Agent Skill
func (sl *SkillLoader) LoadSkill(agentName string) (*SkillConfig, error) {
	if cfg, ok := sl.cache[agentName]; ok {
		return cfg, nil
	}

	filePath, err := sl.findSkillFile(agentName)
	if err != nil {
		return nil, fmt.Errorf("skill not found: %w", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read skill failed: %w", err)
	}

	cfg, err := sl.parseSkillContent(string(content), agentName)
	if err != nil {
		return nil, err
	}

	sl.cache[agentName] = cfg
	return cfg, nil
}

// findSkillFile 在 skills 目录中找到对应的 SKILL.md 文件
func (sl *SkillLoader) findSkillFile(agentName string) (string, error) {
	match, err := filepath.Glob(filepath.Join(sl.skillsDir, "**", "*.SKILL.md"))
	if err != nil {
		return "", err
	}

	for _, f := range match {
		base := filepath.Base(f)
		namePart := strings.TrimSuffix(base, ".SKILL.md")
		namePart = strings.ToLower(namePart)
		if namePart == strings.ToLower(agentName) ||
			namePart == strings.ToLower(strings.Replace(agentName, " ", "", -1)) ||
			strings.Contains(strings.ToLower(base), strings.ToLower(agentName)) {
			return f, nil
		}
	}

	return "", fmt.Errorf("no skill file found for agent: %s", agentName)
}

// parseSkillContent 解析 SKILL.md 文件内容
func (sl *SkillLoader) parseSkillContent(content string, agentName string) (*SkillConfig, error) {
	cfg := &SkillConfig{
		Name:        agentName,
		Metrics: &AgentTraits{
			Name: agentName,
		},
		Content: content,
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	inCharacter := false
	inMetrics := false
	var characterContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "##") {
			section := strings.TrimSpace(strings.TrimPrefix(trimmed, "##"))
			inCharacter = strings.Contains(strings.ToLower(section), "character")
			inMetrics = strings.Contains(strings.ToLower(section), "quality metrics")
			continue
		}

		if inCharacter && trimmed != "" {
			characterContent.WriteString(line + "\n")
		}

		if inMetrics && strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])
				valueStr = strings.TrimLeft(valueStr, "- ")
				switch key {
				case "Efficiency":
					fmt.Sscanf(valueStr, "%f", &cfg.Metrics.Efficiency)
				case "Quality":
					fmt.Sscanf(valueStr, "%f", &cfg.Metrics.Quality)
				case "Creativity":
					fmt.Sscanf(valueStr, "%f", &cfg.Metrics.Creativity)
				case "Collaboration":
					fmt.Sscanf(valueStr, "%f", &cfg.Metrics.Collaboration)
				}
			}
		}
	}

	cfg.Character = characterContent.String()
	if cfg.Character == "" {
		cfg.Character = content
	}

	return cfg, nil
}

// GetSystemPrompt 获取 Agent 的系统提示
func (sc *SkillConfig) GetSystemPrompt() string {
	return sc.Content
}

// SaveSkill 更新 Agent 的 SKILL.md 文件，保存新的特性值
func (sl *SkillLoader) SaveSkill(agentName string, traits *AgentTraits) error {
	// 找到对应的 SKILL.md 文件
	filePath, err := sl.findSkillFile(agentName)
	if err != nil {
		return fmt.Errorf("无法找到 %s 的 SKILL.md 文件: %w", agentName, err)
	}

	// 读取原始文件
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}
	content := string(contentBytes)

	// 替换 Quality Metrics 部分
	newContent := sl.replaceQualityMetrics(content, traits)

	// 写回文件
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("写入 SKILL.md 失败: %w", err)
	}

	// 更新缓存
	if cfg, exists := sl.cache[agentName]; exists {
		cfg.Metrics = traits
	}

	return nil
}

// replaceQualityMetrics 替换 SKILL.md 中的 Quality Metrics 部分
func (sl *SkillLoader) replaceQualityMetrics(content string, traits *AgentTraits) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var builder strings.Builder
	inMetricsSection := false
	metricsFound := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// 检查是否到达 Quality Metrics 章节
		if strings.HasPrefix(trimmed, "##") {
			section := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "##")))
			if strings.Contains(section, "quality metrics") {
				inMetricsSection = true
				// 写入标题
				builder.WriteString(line + "\n")
				// 写入新的 metrics
				builder.WriteString(fmt.Sprintf("- Efficiency: %.2f\n", traits.Efficiency))
				builder.WriteString(fmt.Sprintf("- Quality: %.2f\n", traits.Quality))
				builder.WriteString(fmt.Sprintf("- Creativity: %.2f\n", traits.Creativity))
				if traits.Collaboration > 0 {
					builder.WriteString(fmt.Sprintf("- Collaboration: %.2f\n", traits.Collaboration))
				}
				metricsFound = true
				continue
			} else {
				// 离开 metrics 章节
				inMetricsSection = false
			}
		}

		// 如果不在 metrics 章节，或者还没找到 metrics 章节，正常写入
		if !inMetricsSection || !metricsFound {
			builder.WriteString(line + "\n")
		}
		// 否则跳过旧的 metrics 内容
	}

	return builder.String()
}
