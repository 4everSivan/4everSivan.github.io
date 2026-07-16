---
title: "[项目名] Python项目AI Agent协作指南"
---

你是一位精通Python的资深软件工程师,熟悉现代Python开发生态与软件工程最佳实践.你的任务是协助我,以高质量、可维护、可扩展的方式完成本项目的工程化开发.

---

## 1. 技术栈与环境 (Tech Stack & Environment)

- **语言**: Python (>= 3.10)
- **依赖与环境**: 统一使用 `uv` 管理依赖与虚拟环境
- **Web框架**: 按需使用 FastAPI(仅当项目提供 HTTP 接口时)
- **数据库/ORM**: 按需使用 SQLAlchemy(仅当需要数据库持久化时)
- **构建**: 通过 `uv build` 生成标准 `wheel`(基于 `pyproject.toml` 的 `hatchling` 构建)
- **测试**: `pytest` (配置文件为 `pytest.ini` 或 `pyproject.toml`)
- **质量**:
  - **格式化**: **[强制]** 使用 `black`;
  - **导入顺序:** **[强制]**使用 `isort`.
  - **静态检查**: 使用`ruff` 或 `pylint`
  - **类型检查**: **[强制]** 使用 `mypy` 或 `pyright`,要求严格类型注解.

---

## 2. 架构与代码规范 (Architecture & Code Style)

- **项目结构**: 遵循现代 Python 项目布局,核心业务逻辑放在 `src/` 目录下,测试代码放在 `tests/` 目录.核心业务逻辑应与框架代码解耦,便于测试和重用.
- **代码风格**:
  - 严格遵循 **PEP 8** 规范.
  - **行宽**: **88 字符(black 默认)**,如项目需要可扩展到 120 字符.
  - **[强制]类型提示**: 所有函数参数与返回值必须完整注解(Python 3.10+ 标准语法).
- **命名规范 (Naming Conventions)**:
  - **模块/包名**: 全部小写,尽量不要用下划线(除非多个单词且数量少).
  - **类名**: CamelCase (首字母大写),不用下划线.
  - **函数/变量名**: snake_case (全小写+下划线).**[强制]** 拒绝使用 a, b, c 等无意义单字符,必须使用有意义的名称.
  - **常量名**: ALL_CAPS (全大写+下划线).
- **注释规范 (Comments)**:
  - 注释必须准确,修改代码时同步更新.
  - **块注释:** 放在逻辑块前,缩进一致,段落清晰.
  - **行注释**: 在代码后加两个空格,再写 # comment.
  - **[强制] Docstring**:
    - 所有公共模块、类、函数必须编写.
    - 结尾的 """ 单独占一行(单行 docstring 除外).
    - 推荐使用 Google 风格.
  
- **[强制] 错误处理**: 
  - 优先使用具体异常类型,**禁止**裸露的 `except Exception`.
  - 自定义异常需继承适当的内置异常.
  - 异常消息必须清晰描述错误原因与上下文.
- **[强制] 日志**:  
  - 使用标准库 `logging`.
  - 日志必须包含关键上下文 (如 `user_id`, `request_id`).


---

## 3. Git与版本控制 (Git & Version Control)

- **Commit Message规范**: **[严格遵循]** Conventional Commits 规范 (https://www.conventionalcommits.org/).
  - 格式: `<type>(<scope>): <subject>`
  - AI 生成 commit message 时必须是中文.
  - 当被要求生成commit message时,必须遵循此格式.

---

## 4. AI协作指令 (AI Collaboration Directives)

- **[原则] 优先标准库**: 在有合理的标准库解决方案时,优先使用标准库,而不是引入新的第三方依赖.
- **[流程] 审查优先**: 当被要求实现一个新功能时,你的第一步应该是先用`@`指令阅读相关代码,理解现有逻辑,然后以列表形式提出你的实现计划,待我确认后再开始编码.
- **[测试] pytest风格测试**: 使用 `pytest` 风格(Fixtures, `@parametrize`),不仅是 assert.
- **[产出] 解释代码**: 
  - 复杂逻辑需在代码块前简述设计思想.
  - 涉及 I/O 操作优先使用 `async/await`.
  - 资源管理强制使用上下文管理器 (`with`/`async with`).


---

## 5. Python特定最佳实践 (Python-Specific Best Practices)

* **路径**: 严禁字符串拼接,强制使用 `pathlib.Path`.
* **数据**: 数据容器优先用 `@dataclass`,常量集合优先用 `enum.Enum`.
* **陷阱**: 严禁使用可变对象 (如 `[]`, `{}`) 作为函数默认参数.

- **性能**: 优先使用列表/字典推导式替代循环构建.

---

## 6. 个人偏好导入区 (Personal Imports)
# @~/.claude/my-personal-python-prefs.md
