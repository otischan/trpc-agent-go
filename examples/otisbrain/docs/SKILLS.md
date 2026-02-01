# OtisBrain 技能系统

## 技能系统架构

系统采用标准化的技能组织结构，每个技能都是一个独立的文件夹，包含定义文件和脚本文件：

```
skills/
├── get_recent_critical_events/           # 关键事件查询技能
│   ├── SKILL.md                        # 技能定义文件（固定命名）
│   └── scripts/                        # 技能脚本目录
│       └── get_critical_events.sh      # 获取关键事件的shell脚本
└── get_cluster_resources/              # 集群资源查询技能
    ├── SKILL.md                        # 技能定义文件（固定命名）
    └── scripts/                        # 技能脚本目录
        └── get_cluster_resources.sh    # 获取集群资源的shell脚本
```

这种结构确保了技能的模块化和可维护性，每个技能都可以独立开发、测试和部署。

## 技能系统开发经验总结

### 1. 技能定义格式
- 使用 SKILL.md 文件定义技能，包含 YAML frontmatter 和 Markdown 内容
- frontmatter 部分定义技能元数据（name, description 等）
- body 部分包含参数说明、示例命令和输出说明
- 正确的 frontmatter 格式为：
  ```
  ---
  name: skill-name
  description: skill description
  ---
  ```

### 2. 路径配置注意事项
- 日志文件实际写入路径：`logs/basic/critical_events.log`
- 聚合后摘要文件路径：`logs/critical_record/critical_summary_YYYYMMDD_HHMMSS.txt`
- 技能脚本只使用聚合后的摘要文件，不回退到原始日志文件
- 使用 `ls -t` 命令按时间排序查找最新的摘要文件
- 聚合文件包含更高质量的信息，仅使用聚合文件进行查询反馈

### 3. 正则表达式处理
- SKILL.md 文件解析需处理不同操作系统的换行符（LF vs CRLF）
- 使用灵活的正则表达式 `[\r\n]` 来匹配不同换行符
- 确保 frontmatter 解析正则表达式能正确匹配分隔符

### 4. Shell 脚本编写
- 脚本应包含适当的错误处理和回退机制
- 使用参数化方式处理输入参数
- 脚本需具有可执行权限（chmod +x）
- 脚本路径应相对于技能目录的 scripts 子目录

### 5. 参数模板处理
- 在 SKILL.md 中使用 `{{ inputs.param_name }}` 语法定义参数占位符
- 系统会自动将此语法转换为 Go 模板语法 `{{.param_name }}`
- 参数在运行时会被实际值替换

### 6. 技能执行机制
- 技能通过通用的 shell 命令执行机制运行，实现与代码的完全解耦
- 支持 `shell_command` 和 `function_call` 两种执行类型
- 通过 `list skills` 命令可动态查看可用技能列表

## 使用场景

### 聊天服务应用场景

1. **实时状态查询**: 用户可随时询问集群当前状态
2. **问题诊断咨询**: 与AI助手对话，获取问题诊断和解决建议
3. **运维操作指导**: 获取运维操作建议和最佳实践
4. **异常信息查询**: 查询近期发生的异常事件和处理情况
5. **技能化操作**: 通过自然语言调用预定义技能，如查询关键事件、资源使用情况等