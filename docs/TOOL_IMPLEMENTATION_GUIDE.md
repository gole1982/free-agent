# 工具最优实现方案总结

## 各工具实现方式对比

| 工具 | 最优方式 | 输出格式 | 结构化程度 | Agent友好度 |
|------|---------|---------|-----------|------------|
| **sqlmap** | REST API (`sqlmapapi.py`) | JSON | ★★★★★ | ★★★★★ |
| **OWASP ZAP** | REST API | JSON | ★★★★★ | ★★★★★ |
| **Burp Suite** | MCP 调用 | JSON | ★★★★★ | ★★★★★ |
| **Nmap** | XML输出 (`-oX`) | XML | ★★★★☆ | ★★★★☆ |
| **Nikto** | JSON输出 (`-Format json`) | JSON | ★★★★☆ | ★★★★☆ |
| **Metasploit** | RPC API | JSON | ★★★★★ | ★★★★★ |

---

## 1. sqlmap - REST API 方式

### 启动API服务
```bash
python sqlmapapi.py -s -p 8775
```

### API调用流程
```
1. POST /task/new          → 获取taskID
2. POST /option/{taskID}/set → 设置目标URL
3. GET  /scan/{taskID}/start → 启动扫描
4. GET  /scan/{taskID}/status → 轮询状态
5. GET  /scan/{taskID}/data  → 获取结果(JSON)
6. GET  /task/{taskID}/delete → 删除任务
```

### 结构化输出示例
```json
{
  "data": [
    {
      "type": "Boolean-based blind",
      "parameter": "id",
      "place": "GET",
      "payload": "1 AND 1=1",
      "dbms": "MySQL",
      "title": "MySQL boolean-based blind"
    }
  ]
}
```

### 实现文件
- [sqlmap_api_adapter.go](file:///d:/Programing/free-agent/internal/vds/tools/sqlmap_api_adapter.go)

---

## 2. OWASP ZAP - REST API 方式

### 启动API服务
```bash
zap.sh -daemon -port 8080
```

### API调用流程
```
1. GET /JSON/ascan/action/scan?url={target} → 启动扫描
2. GET /JSON/ascan/view/status?scanId={id} → 获取进度
3. GET /JSON/alert/view/alerts?baseurl={target} → 获取警报
```

### 结构化输出示例
```json
{
  "alerts": [
    {
      "name": "Cross Site Scripting (Reflected)",
      "risk": "High",
      "description": "...",
      "solution": "...",
      "url": "http://target.com/search?q=test",
      "param": "q",
      "evidence": "<script>alert(1)</script>",
      "cweid": "79",
      "wascid": "8"
    }
  ]
}
```

### 实现文件
- [zap_api_adapter.go](file:///d:/Programing/free-agent/internal/vds/tools/zap_api_adapter.go)

---

## 3. Burp Suite - MCP 调用方式

### MCP工具调用
```go
mcpClient.CallTool("mcp_burp_send_http1_request", {
    "url": "http://target.com",
    "method": "GET"
})
```

### 结构化输出
- Burp Suite MCP 返回标准化的JSON结果
- 包含请求/响应详情、漏洞信息

---

## 4. Nmap - XML输出方式

### 命令
```bash
nmap -sV -sC -oX - target.com
```

### XML结构
```xml
<nmaprun>
  <host>
    <address addr="192.168.1.1" addrtype="ipv4"/>
    <ports>
      <port protocol="tcp" portid="80">
        <state state="open"/>
        <service name="http" product="Apache" version="2.4.41"/>
        <script id="http-vuln-cve2021-41773" output="VULNERABLE"/>
      </port>
    </ports>
  </host>
</nmaprun>
```

### 实现文件
- [nmap_xml_adapter.go](file:///d:/Programing/free-agent/internal/vds/tools/nmap_xml_adapter.go)

---

## 5. Nikto - JSON输出方式

### 命令
```bash
nikto -h target.com -Format json -output result.json
```

### JSON结构
```json
{
  "vulnerabilities": [
    {
      "id": "OSVDB-0",
      "method": "GET",
      "url": "/admin",
      "msg": "Admin directory found"
    }
  ]
}
```

---

## 6. Metasploit - RPC API 方式

### 启动RPC服务
```bash
msfrpcd -P password123 -S
```

### API调用
```go
POST /api/v1/exploit
{
  "module": "exploit/multi/http/apache_mod_cgi_bash_env_exec",
  "target": "http://target.com"
}
```

---

## 结构化结果对Agent的价值

### ✅ 结构化输出优势

| 优势 | 说明 |
|------|------|
| **自动解析** | 无需硬编码字符串匹配 |
| **精确信息** | 参数名、payload、DBMS等精确提取 |
| **下一步操作** | Agent可直接使用漏洞信息进行利用 |
| **报告生成** | 自动生成标准报告 |
| **漏洞链构建** | 多个漏洞可自动关联 |

### 示例：Agent使用结构化结果

```go
// Agent收到结构化漏洞发现
finding := &VulnerabilityFinding{
    Type: "SQL Injection (Boolean-based blind)",
    Location: "Parameter: id, Place: GET",
    Payload: "1 AND 1=1",
    DBMS: "MySQL",
    Exploitable: true,
}

// Agent可以：
// 1. 自动选择利用模块
// 2. 构造精确的利用payload
// 3. 执行数据库枚举
// 4. 生成修复建议
```

---

## 实现优先级

| 优先级 | 工具 | 原因 |
|-------|------|------|
| **P0** | sqlmap API | SQL注入最常见，API最成熟 |
| **P0** | ZAP API | Web扫描全覆盖，API完整 |
| **P1** | Nmap XML | 网络扫描基础，XML稳定 |
| **P1** | Burp MCP | 专业渗透测试，MCP集成 |
| **P2** | Nikto JSON | Web服务器扫描补充 |
| **P2** | Metasploit RPC | 高级利用阶段 |