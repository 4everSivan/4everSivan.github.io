---
title: "Patroni版本日志"
lastmod: "2023-08-09T17:02:32+08:00"
---

[TOC]

## Version 3.1.0

### 重大更改

* **更改了 `restapi.keyfile` 和 `restapi.certfile` 的语义（Alexander Kukushkin）**

  之前，如果在节中没有相应的配置参数，Patroni 会将 `restapi.keyfile` 和 `restapi.certfile` 作为客户端证书的后备选项。

>  **警告：**
>
> * 如果启用了客户端证书验证（`restapi.verify_client` 设置为 `True`），还必须在 `restapi.certfile`、`restapi.keyfile` 和 `restapi.keyfile_password` 中提供有效的客户端证书。如果未提供，Patroni 将无法正确工作。

### 新功能

* **使 Pod 角色标签可配置（Waynerv）**

  可以使用 `kubernetes.leader_label_value`、`kubernetes.follower_label_value` 和 `kubernetes.standby_leader_label_value` 参数自定义值。当我们将角色更改为 `master` 时，此功能将非常有用。您可以在此处阅读有关该功能和迁移步骤的更多信息：:ref:`在这里 <kubernetes_role_values>`。

### 改进

* **对 `patroni --validate-config` 进行了各种改进（Alexander Kukushkin）**

  增强了不同 DCS、`bootstrap.dcs`、`ctl`、`restapi`、`watchdog` 等部分的参数验证。

* **在 Patroni 运行时，如果在恢复期间崩溃的话，启动 PostgreSQL 时不再处于恢复状态（Alexander Kukushkin）**

  这可能会减少恢复时间，并有助于防止不必要的时间线增量。

* **避免不必要的 `/status` 键更新（Alexander Kukushkin）**

  当主库上没有永久的逻辑复制槽时，Patroni 在每个心跳循环中更新 `/status` 键，即使主库上的 LSN 没有前进。

* **不允许陈旧的主服务器赢得领导者竞争（Alexander Kukushkin）**

  如果由于资源不足而导致 Patroni 长时间挂起，它还会在获取领导者锁之前额外检查是否有其他节点在提升 PostgreSQL 之前已成为主服务器。

* **实现了某些 PostgreSQL 参数验证的可见性（Alexander Kukushkin，Feike Steenbergen）**

  如果 `max_connections`、`max_wal_senders`、`max_prepared_transactions`、`max_locks_per_transaction`、`max_replication_slots` 或 `max_worker_processes` 的验证失败，Patroni 将使用一些合理的默认值。现在除此之外，它还将显示一个警告。

* **为在 `PGDATA` 中创建的文件和目录设置权限（Alexander Kukushkin）**

  之前，Patroni 创建的所有文件只有所有者的读写权限。这种行为破坏了在不同用户下运行的备份工具，因为这些工具依赖于组的读权限。现在，Patroni 尊重和正确设置所有内部创建的目录和文件的权限。

### 错误修复

* **通过 shell 运行 archive_command (Waynerv)**

  在单用户模式下或在进行崩溃恢复之前，Patroni 可能会通过 shell 运行一些 WAL 段，或者在执行 pg_rewind 之前。如果 archive_command 包含一些 shell 操作符，比如 &&，那么在 Patroni 中可能无法正常工作。

* **修复 "on switchover" 关闭检查问题 (Polina Bungina)**

  指定的候选人可能仍在流式传输，并且尚未接收关闭检查，但是领导者键已被删除，因为其他节点是健康的。

* **修复 "is primary" 检查问题 (Alexander Kukushkin)**

  在领导者竞争过程中，副本可能无法识别旧领导者上的 Postgres 是否仍在以主模式运行。

* **修复 patronictl list (Alexander Kukushkin)**

  在 tsv、json 和 yaml 输出格式中，集群名称字段缺失。

* **修复在暂停后的 pg_rewind 行为 (Alexander Kukushkin)**

  在某些条件下，Patroni 在维护模式结束后，可能无法使用 pg_rewind 将错误的主节点重新加入集群。

* **修复 Etcd v3 实现中的错误 (Alexander Kukushkin)**

  如果使用 `create_revision/mod_revision` 字段执行密钥更新，由于修订版本不匹配，会使内部 KV 缓存失效。

* **修复暂停状态下的备份集群中副本的行为 (Alexander Kukushkin)**

  当领导者键过期时，处于备份集群中的副本将不会跟随远程节点，而是保持 primary_conninfo 不变。

## Version 3.0.4
### 新功能

* **使备用节点的复制状态可见 (Alexander Kukushkin)**

  对于 PostgreSQL 9.6+，Patroni 将报告备用节点的复制状态，当备用节点正在从其他节点进行流复制时，状态将显示为 `streaming`；当没有复制连接且设置了 `restore_command` 时，状态将显示为 `archive recovery`。这个状态在分布式协调服务（DCS）的 `member` 键、`REST API` 和 `patronictl list` 输出中都是可见的。

### 改进

* **改进 Etcd v3 的错误消息（Alexander Kukushkin）**

  当 Etcd v3 集群无法访问时，Patroni 会报告无法访问 `/v2` 端点的错误。

* **在可能的情况下，在 patronictl 中使用法定读取（Alexander Kukushkin）**

  Etcd 或 Consul 集群可能降级为只读状态，但从 patronictl 视图来看一切都正常。现在它将失败并显示错误信息。

* **防止配置中出现重复的名称引发分裂脑问题（Mark Pekala）**

  在启动 Patroni 时，它会检查是否在 DCS 中注册了相同名称的节点，并尝试查询其 REST API。如果 REST API 可访问，Patroni 将退出并显示错误。这有助于防止人为错误。

* **如果在 Patroni 运行时崩溃，不再将 Postgres 启动到恢复模式（Alexander Kukushkin）**

  这可能会减少恢复时间，并有助于避免不必要的时间轴增量。

### 错误修复

* **在收到 SIGHUP 信号时未重新加载 REST API SSL 证书（Israel Barth Rubio）**

  在 3.0.3 版本中引入了回归问题。

* **修复参数像 max_connections 这样的整数 GUC 验证（Feike Steenbergen）**

  Patroni 不支持带引号的数字值。在 3.0.3 版本中引入了回归问题。

* **修复 synchronous_mode 问题（Alexander Kukushkin）**

  使用 `synchronous_commit=off` 执行 `txid_current()`，以避免在启用 `synchronous_mode_strict` 时意外等待缺少同步备份。

## Version 3.0.3
### 新功能

* **与 PostgreSQL 16 beta1 的兼容性（Alexander Kukushkin）**

  扩展了 GUC（全局用户配置）的验证规则。

* **使 PostgreSQL GUC 的验证器可扩展（Israel Barth Rubio）**

  验证规则从 `patroni/postgresql/available_parameters/` 目录中的 YAML 文件加载。文件按字母顺序排列，依次应用。这使得可以为非标准的 Postgres 发行版编写自定义验证器。

* **新增 restapi.request_queue_size 选项（Andrey Zhidenkov, Aleksei Sukhov）**

  设置 Patroni REST API 使用的 TCP 套接字的请求队列大小。一旦队列已满，进一步的请求将获得`Connection denied` 错误。默认值为 `5`。

* **在初始化新集群时直接调用 initdb（Matt Baker）**

  以前是通过 `pg_ctl` 调用的，需要对传递给 `initdb` 的参数进行特殊引用。

* **新增停止前钩子（Le Duane）**

  该钩子可以通过 `postgresql.before_stop` 配置，并在 `pg_ctl stop` 之前执行。退出代码不影响关闭过程。

* **添加自定义 Postgres 二进制文件名的支持（Israel Barth Rubio, Polina Bungina）**

  在使用自定义的 Postgres 发行版时，Postgres 二进制文件可能与社区 Postgres 发行版使用的名称不同。可以使用 `postgresql.bin_name.*` 和 `PATRONI_POSTGRESQL_BIN_*` 环境变量来配置自定义的二进制文件名。

### 改进

* **patroni --validate-config 的各种改进（Polina Bungina）**
  * 使 `bootstrap.initdb` 可选。它仅对新集群需要，但是如果在配置中缺少它，`patroni --validate-config` 会报错。
  * 当 `postgresql.bin_dir` 为空或未设置时，不要报错。首先尝试在默认路径中查找 Postgres 二进制文件。
  * 将 `postgresql.authentication.rewind` 部分设为可选。如果缺少此部分，Patroni 将使用超级用户。

* **在 patronictl 中改进错误报告（Israel Barth Rubio）**

  `\n` 符号被渲染为原样，而不是实际的换行符。

### 错误修复

* **修复了 Citus 支持中的问题（Alexander Kukushkin）**

  如果在 switchover 过程中，从被提升的 worker 到协调器的 REST API 调用失败，会导致该 Citus 组被无限期地阻塞。

* **允许在 patronictl 的 --dcs-url 选项中使用 etcd3 URL（Israel Barth Rubio）**

  如果用户尝试通过 patronictl 的 `--dcs-url` 选项传递 etcd3 URL，将会遇到异常。

## Version 3.0.2
> **警告: **
>
> * 版本 3.0.2 不再支持 Python 3.6 之前的版本。

### 新功能

* **向 /metrics 端点添加了同步的备用副本状态 (Thomas von Dein, Alexander Kukushkin)**

  以前只报告 `primary/standby_leader/replica`。

* **在 patronictl 中更友好地处理 PAGER (Israel Barth Rubio)**

  它通过 PAGER 环境变量使 `pager` 可配置，覆盖默认的 `less` 和 `more`。

* **使 K8s 可重试的 HTTP 状态码可配置 (Alexander Kukushkin)**

  在某些托管平台上，可能会获得状态码 `401` 未经授权，而这个问题有时在几次重试后会解决。

### 改进

* **在自定义引导过程中，只有在 recovery_target_action 设置为 promote 时才将 hot_standby 设置为 off (Alexander Kukushkin)**

  这是为了确保 `recovery_target_action=pause` 正确工作。

* **不允许 on_reload 回调干扰其他回调函数 (Alexander Kukushkin)**

  `on_start/on_stop/on_role_change` 通常用于添加/移除虚拟 IP，而 `on_reload` 不应干扰它们。

* **在 AWS 回调示例脚本中切换到 IMDSFetcher (Polina Bungina)**

  `IMDSv2` 需要一个令牌来进行工作，而 `IMDSFetcher` 会透明地处理它。

### 错误修复

* **修复了在运行在 Kubernetes 上的 Citus 集群中的 patronictl switchover 问题 (Lukáš Lalinský)**

  在与默认命名空间不同的命名空间中无法正常工作。

* **如果未知主版本号，则不写入 PGDATA (Alexander Kukushkin)**

  如果在启动后 `PGDATA` 为空（可能尚未挂载），Patroni 将错误地假设 PostgreSQL 版本，并错误地创建 `recovery.conf` 文件，即使实际的主版本号是 `v10+`。

* **修复了协调器故障转移后 Citus 元数据的 bug (Alexander Kukushkin)**

  `citus_set_coordinator_host()` 调用不会引起元数据同步，导致工作节点上的更改不可见。通过切换到 `citus_update_node()` 解决了这个问题。

* **在所有 etcd 节点“失败”时，使用配置文件中列出的 etcd 主机作为后备 (Alexander Kukushkin)**

  etcd 集群的拓扑可能随时间改变，Patroni 试图跟随变化。如果在某一时刻所有节点都不可达，Patroni 将在尝试重新连接时使用配置文件中的节点与最后已知的拓扑结合。

## Version 3.0.1

### 错误修复

- 将正确的角色名称传递给 `on_role_change` 回调脚本 (Alexander Kukushkin, Polina Bungina)

  在升级过程中，Patroni 错误地将 `promoted` 角色传递给了 `on_role_change` 回调脚本。现在，传递的角色名称已恢复为 `master`。这个回归错误是在版本 3.0.0 中引入的。

## Version 3.0.0

这个版本增加了与 [Citus](https://www.citusdata.com/) 的集成，并使得在临时的 DCS 中断情况下能够在不降级主节点的情况下存活。

> **警告:**
>
> - 版本 3.0.0 是最后一个支持 Python 2.7 的版本。未来的发布将不再支持低于 3.7 版本的 Python。
> - RAFT 支持已被弃用。我们会尽力维护它，但不对可能出现的问题提供任何保证或责任。
> - 这个版本是摒弃 "master"，采用 "primary" 的第一步。只有运行至少 3.0.0 版本的情况下，升级到下一个主要版本才能正常运行。

### 新功能

- **DCS 安全模式 (Alexander Kukushkin, Polina Bungina)**

  如果启用此功能，Patroni 集群将能够在临时的 DCS 中断情况下存活。您可以在 [:ref:`文档`](https://github.com/zalando/patroni/blob/master/docs/releases.rst#id3) 中找到更多详细信息。

- **Citus 支持 (Alexander Kukushkin, Polina Bungina, Jelte Fennema)**

  Patroni 使得轻松部署和管理具有高可用性的 [Citus](https://www.citusdata.com/) 集群成为可能。请查看 [:ref:`这里`](https://github.com/zalando/patroni/blob/master/docs/releases.rst#id5) 页面以获取更多信息。

### 改进

- **抑制删除未知但活动的复制槽时的重复错误 (Michael Banck)**

  Patroni 仍会记录这些日志，但只会在调试模式下记录。

- **每个 HA 循环只运行一个监控查询 (Alexander Kukushkin)**

  如果启用了同步复制，这就不是问题了。

- **只保留最新的失败数据目录 (William Albertus Dembo)**

  如果引导失败，以前 Patroni 会将 `$PGDATA` 文件夹重命名为时间戳后缀。从现在开始，后缀将是 `.failed`，如果这样的文件夹存在，则在重命名之前会将其删除。

- **改进同步复制连接的检查 (Alexander Kukushkin)**

  当新主机添加到 `synchronous_standby_names` 时，只有在它在 `pg_stat_replication.sync_state = 'sync'` 的基础上成功追赶上主数据库后，它才会在 DCS 中被设置为同步。

### 已移除功能

- **移除 `patronictl scaffold` (Alexander Kukushkin)**

  其唯一用途是以一个不稳定的方式运行备用集群。
