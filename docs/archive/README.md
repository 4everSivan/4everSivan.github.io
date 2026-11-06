# 版本与阶段归档库 (archive/)

> **created**: 2026-11-06 ｜ **last-change**: 2026-11-06 ｜ **status**: active

## 1. 简介

本目录封存已发布版本或已收口阶段的 C/T 卡片。按真实发布批次归档，卡片内容、状态与本批次路由一起冻结：现行总账只登记活跃卡，归档卡由所在批次的批次总账路由，历史与依赖查询经批次总账完成。

归档属于已授权的发版或阶段收口操作；卡片完成不会自动授权打 Tag、推送或部署。现行设计继续在 `docs/devel/design/` 维护。

## 2. 索引

| 入口或位置 | 保存内容 | 维护方式 |
|---|---|---|
| [变更现行总账](../devel/change/index.json) | 活跃 C 卡、Topic 与设计正文的对应 | 建卡与转状态时维护；归档时移出本批次登记 |
| [任务现行总账](../devel/task/index.json) | 活跃 T 卡、阶段与依赖 | 建卡与转状态时维护；归档时移出本批次登记 |
| `<version>/index.json` | 本批次总账（`adflow-archive-batch-v1`）：C/T 卡路由与主题归属 | 归档时新建，之后只读 |
| `<version>/change/` | 本批次 `closed` C 卡 | 归档后只读 |
| `<version>/task/` | 本批次 `completed` T 卡 | 归档后只读 |

### 归档批次清单

| 项目版本或阶段 | 归档路径 | 归档日期 | 范围说明 |
|---|---|---|---|

完成实际归档后登记本表；空项目不预建示例版本或填入示例卡片。

## 3. 归档规则

- **范围明确**：按卡片 `version` 和本次发布范围选择卡片，确认本批次 C 卡均 `closed`、T 卡均 `completed`；其他批次卡和未完成事项留在活跃区。
- **前提齐备**：以 `scripts/adflow-verify --release <version>` 核定本批次全部活跃卡为终态并取得 Exit 0（普通校验 Exit 0 本身不证明批次终态——非终态活跃卡没有状态前置要求）；未填 `version` 的未收口卡先划定批次；预检以 `RELEASE_STRANDED` 提示批次早于本次发布、仍滞留活跃区的终态卡，由人工决定是否随本批次归档（卡片 `version` 保持原值）。相关构建、测试及项目要求的发布验收完成，todo 已评审。
- **冻结内容**：移动卡片不改写其状态、验收、提交或 `gate`；唯一例外是第 3 步随归档提交填写的 `sync.tag`。后续补充与修正使用现行设计或新卡表达，不重写历史卡。
- **Tag 最后打**：Tag 指向的提交必须已包含归档移动、总账更新与定稿的 CHANGELOG；先打 Tag 再补归档或 CHANGELOG 会让 Tag 与发布记录不一致。
- **分层路由**：本批次卡片的登记条目与主题归属随卡片迁入 `<version>/index.json` 并冻结，现行总账删除这些登记，`topics[*].design_doc` 继续保留，设计正文的总账链接不变。卡号不复用，新卡号以 `scripts/adflow-verify --next-id change|task` 覆盖全部批次计算。旧项目在现行总账中保留归档登记仍然有效（`LEDGER_ARCHIVED_ENTRIES` 旁路提示），可按第 6 节瘦身。
- **保护目的地**：移动前核对每个目的路径；已有同名文件时先查明冲突，禁止覆盖。归档动作本身不要求改动设计正文或重编号。

## 4. 发版归档四步

所有路径相对于项目根目录；`<version>` 替换为真实发布批次。

1. **核定范围并建立目录**：列明本次卡号与原路径，运行批次预检取得 Exit 0，检查目的路径后建立 `docs/archive/<version>/change/` 和 `task/`；不移动其他批次内容。

```bash
scripts/adflow-verify --release <version> --json
```

2. **移动终态卡片**：将核定的 C/T 卡移动到对应轨道目录，核对文件数量与内容完整性，不覆盖现有文件。
3. **写入批次总账、更新现行总账并定稿 CHANGELOG**：新建 `docs/archive/<version>/index.json`，声明 `"$schema": "adflow-archive-batch-v1"` 并填写 `release`、`archived_at`；把本批次卡片的登记条目原样移入其 `cards` / `tasks`，`file` 改为实际归档路径（如 `docs/archive/<version>/task/T01.json`），把各卡在现行 `topics` 中的归属移入清单同名主题。随后从现行总账删除这些登记与主题引用，保留 `topics[*].design_doc`，完成后登记第 2 节「归档批次清单」表。根目录 CHANGELOG 的 `[Unreleased]` 改为 `[<version>] - <发布日期>` 并新建空的 `[Unreleased]`，本批次卡片链接改指归档路径；项目使用 `sync.tag` 时为本批次 C 卡填写即将打出的 Tag 名。
4. **复核、提交并打 Tag**：再次运行普通校验，以 `scripts/adflow-verify --topic <主题>` 从设计主题定位归档卡，核对路径、卡号、状态与依赖可达，且没有新增 `ARCHIVE_UNREGISTERED`；确认本次归档未改变设计正文或除 `sync.tag` 外的卡片内容。归档区出现非终态卡将被判 `ARCHIVE_NON_TERMINAL` 确定性违规（Exit 4），必须移回活跃区或先完成收口，禁止以改状态方式静默"补票"。Exit 0 后提交本次归档、总账与 CHANGELOG 变更，在发布已获授权时于该提交上打 Tag；推送另需授权。

```bash
scripts/adflow-verify --json
git tag -a <version> -m "Release <version>"
```

后续任务可经批次总账只读使用归档依赖的 `completed` 状态。校验器核对归档文件存在、身份、状态一致性，并要求归档区只含终态卡（非 `closed`/`completed` 即 `ARCHIVE_NON_TERMINAL`）；未经现行总账或批次总账登记、直接放入归档目录的卡同样受检，并以 `ARCHIVE_UNREGISTERED` 提示补登路由。校验器不按当前业务规则重验或回写归档卡的 `gate` 与证据字段，归档后的 Exit 0 也不能替代归档前的业务验收。

中断时保留已完成移动及现有记录，按实际路径修复剩余路由，不批量重复移动。构建产物、运行数据和临时报告仍留在 `local/`，不以归档名义清理 `local/` 或 `_adflow_backup/`。

## 5. 历史批次账实修复

升级到新版治理规范后，历史归档区中的非终态卡会被普通校验暴露为 `ARCHIVE_NON_TERMINAL`（Exit 4）。这是对既有归档前置条件的机器覆盖，属预期暴露而非升级引入的故障；**改写任何归档卡状态都必须由人工治理口径驱动，AI 不得代办**。流程：

1. **盘点**：运行 `scripts/adflow-verify --json` 列出全部发现，逐卡对照 Git Tag、CHANGELOG、git 历史与运行事实，核实该卡工作是否已随版本实际交付；
2. **人工定口径**（每卡三选一）：**补收**（已交付：状态改 `closed`/`completed`，卡内追加 `closure` 豁免标注注明人工决策依据与佐证，严禁伪造 `user_quote`/`signoff`/`gate`/`sync.commit`，缺失即留空由标注说明）；**回迁**（未完成且事项仍有效：移回活跃卡池按正常双轨流程推进，自然过门禁留真实 gate）；**废弃**（未完成且已无意义：人工判定后置终态并注明废弃依据）；
3. **修正派生矛盾**：总账或批次总账的状态字段与卡片同步，里程碑状态与成员卡对齐，topics 登记核对；
4. **勘误登记**：在本批次归档目录建勘误记录（如 `ERRATUM.md`）并登记上方批次清单：事故描述、涉及卡号、处置口径与日期、审计声明（该版本验收以 tag、CHANGELOG 与运行事实为准，卡片状态系人工口径修复，非当时真实收口）；
5. **（推荐）治理修复建卡**：本次修复本身按正常变更流建一张 C 卡走完 `verified`→`closed`，人工定口径的会话原话即合规 `user_quote`；
6. **复核收口**：普通校验恢复 Exit 0、`--mode init` 通过，CHANGELOG 记录本次治理修复。

## 6. 现行总账瘦身

旧版规范要求现行总账保留全部历史登记，现行总账因此随卡片总数线性膨胀。升级后校验器以聚合的 `LEDGER_ARCHIVED_ENTRIES` 旁路提示仍留在现行总账中的归档登记；旧布局继续有效，规范同步不自动迁移。需要瘦身时由人工授权一次性执行，**不改写、不移动任何卡片文件**：

1. **保存对照基线**：在干净工作区运行 `scripts/adflow-verify --json`、`--next-id change`、`--next-id task`，并对各主题运行 `--topic <主题> --json`，保存输出；
2. **建立批次总账**：按现行登记 `file` 所在的归档目录分组，为每个批次新建 `docs/archive/<version>/index.json`（同第 4 节第 3 步格式，`release` 取目录名，`archived_at` 取第 2 节「归档批次清单」表的日期）；登记条目原样移入，现行 `topics` 中指向这些卡的归属移入清单同名主题。目的文件已存在时先查明，禁止覆盖；项目自建的历史快照（如 `<version>/change/index.json`）保持原样；
3. **删除现行登记**：从现行总账删除已迁移的条目与主题引用，保留 `topics[*].design_doc`、`milestones` 与全部活跃登记；
4. **等价复核**：重跑第 1 步命令——除 `LEDGER_ARCHIVED_ENTRIES` 消失外，发现清单与退出码不变，`--next-id` 结果不变，各主题 `--topic` 结果不变，且无新增 `ARCHIVE_UNREGISTERED`。任一不等即停止并恢复本次改动，不放宽检查；
5. **（推荐）治理建卡**：本次瘦身按正常变更流建一张 C 卡走完 `verified`→`closed`，并在 CHANGELOG 记录。
