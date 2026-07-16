---
title: "Patroni 同步复制"
---

## Patroni 处理同步复制

同步复制模式需要在Patroni配置文件中设置 `synchronous_mode = True`  。

在发生主备切换/故障转移时，不同角色的 patroni 节点的处理方式如下：

1. master, primary, promoted

   ![img](https://gitee.com/sivan819/img_repo/raw/master/img/patroni%20process_sync_replication%20%E6%B5%81%E7%A8%8B%E5%9B%BE.drawio.png)

2. replica

3. demoted

4. standby_leader

5. uninitialized
