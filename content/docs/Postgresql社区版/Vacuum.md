---
title: "Vacuum"
lastmod: "2024-12-20T09:17:34+08:00"
---

## 1. 概念

### 1.1 VACUUM 真空清理

VACUUM 用于回收死元组占用的存储空间。这些死元组是由于通过更新过期或者删除的元组不会从表中进行物理移除，直到执行一个 VACUUM 操作完成后才会被从表对应的物理文件中移除。因此在频繁更新的表上需要定期执行 VACUUM 操作。

`VACUUM` 命令有两种用法：

`VACUUM FULL`：回收磁盘空间并重新组织表中的数据，但不删除已删除的行。

`VACUUM SORT ONLY`：重新组织数据，但不回收磁盘空间或删除已删除的行。



`VACUUM` 处理表的目的：

* 恢复或者重用由UPDATE 和 DELETE 删除元组后占用的磁盘空间

* 更新由 PostgreSQL 查询计划器使用的数据统计信息

* 更新可见性映射，可以加快 index-only scan 的速度

* 保护由于事务 ID 循环处理或者多个事务ID循环处理导致老旧数据的丢失

### 1.2 可见性映射（VM）

VM 是可见性映射文件表，每一个表都有可见性映射文件，以跟踪页面中包含已知对所有活跃事务可见的元组，同时也跟踪页面仅包含冻结的元组。该文件命名以表对象 oid作为前缀，`_vm` 作为后缀存在。

##### 可见性映射位

可见性映射是一个位图，每个 `heap page` 都有两个位，表示全部页面可见和全部页面冻结。如果可见位被设置，意味着页面上所有元组对于所有事务都是可见的，因此该页面不需要清理。如果冻结位被设置，意味着页面上所有的元组都被完全冻结，因此即使需要对整个表进行 `VACUUM`（如: `anti-wraparound` ），也无需要对该页面进行 `VACUUM` 处理。仅当页面全部已经可见时，才需要对全冻结位设置。

```c
src/include/access/visibilitymap.h
/* 每个堆页面上的位数定义*/ 
#define BITS_PER_HEAPBLOCK 2

/* 位映射标识 */ 
#define VISIBILITYMAP_ALL_VISIBLE 0x01 
#define VISIBILITYMAP_ALL_FROZEN  0x02 
#define VISIBILITYMAP_VALID_BITS  0x03 /*所有可见的位映射标识*/
```

清除可见性映射位不会单独进行 WAL 记录。调用者只要确认位被清除，那么 WAL 在重放时，位也将被清除。如果没有设置任何位，有可能或者当在 VACUUM 期间设置一个可见性映射时，必须要写入 WAL。页面本身上的 `PD_ALL_VISIBLE` 位和可见性映射位被放置一起。因此，如果可见性映射页写入到磁盘后和更新堆页面在写入磁盘之前，如果实例发生崩溃，该堆页面上该位必须被重置（ redo )。否则在堆页面上进行下一次 INSERT、UPDATE 或者 DELETE 将无法知道必须清除可见性映射位，从而导致索引扫描返回错误。

##### 如何清除位

设置位时，需要在堆页面上保持锁定，这样可以防止出现竞争情形，即VACUUM 看到页面上的所有元组对所有人都可见，但另外一个后端会在 VACUUM 设置可见性映射位之前修改页面。当位已被设置后，将更新可见性映射页面的LSN，以确保在刷新可以设置该位的 WAL 记录之前，不会将可见性视图更新写入磁盘。当位被清除后，就不需要该操作了。

##### 如何检查位——pg_visibility

该扩展提供检查表的可见性映射和页面级可见性信息的方式，同时提供了对可见性映射进行强制并对其进行重建的函数。三个不同的位通常用来存储页级可见性信息。VISIBILITYMAP_ALL_VISIBLE 位表明表页面中的每一个元组对于当前和将来的事务都是可见的。VISIBILITYMAP_ALL_FROZEN 位表明表页面中的所有元组是被冻结的，即在元组被INSERT,UPDATE,DELETE或者锁定该页面前，将来不在需要修改的页面。页头中的PD_ALL_VISIBLE 位与 ALL_VISIBLE位相同，只不过其存储在数据页本身。

### 1.3 冻结（FREEZE）

冻结过程有两种：一种是lazy 冻结，一种是 aggressive 冻结。

* lazy：冻结过程仅使用表对应的空闲空间映射文件中包含死元组的页面。
* aggressive：冻结过程会对表的整个页面进行扫描。无论该表页面中是否包含有死元组。并且在可能的情况下才会移除 xact（clog）文件。

源码相关参数：

* `oldestXmin`：用来区分元组是 `DEAD` 或者是 `RECENTLY_DEAD` 的截断值。

* `frozenLimit`：在 VACUUM 期间，所有的 `xids` 低于该 `xid` 的都将使用 `frozenLimit` 替换，即冻结该 `xid`。

* `xidFullScanLimit`：根据 `table_freeze_age` 参数计算，表示最小的xid值。若 `relforzenxid` 大于该表的表都会被 VACUUM，以此来冻结整个表中的元组。小于此值的表只扫描VM包含死元组的页面。

* `multiXactCutoff`：对低于该值的 `xid`，从 `xmax` 中移除所有的 `multixactids`。

* `mxactFullScanLimit`：与 `xidFullScanlimit` 类似。

了解了上面的相关参数，引出了 `freezeLimit_txid` 的计算公式：

```c
limit = *oldestXmin - freezemin;
```

其中 `oldestXmin` 为当前运行事务中最早的事务标识。如有 T1，T2，T3 三个事务，xid分别为 100、101 和 102，那么最早的事务的ID应该为 `T1=100`。`freezemin` 数据库默认值为 50000000。

#### 1.3.1 lazy 模式下的冻结

假设当前最早的事务标识大于 `freezemin`，即设定为 `50000800`，那么在该模式下冻结的 limit 应该为 `50000800 – 50000000` ，即 limit 为 800。

假设位于 0 号块在 tuple1 上行片头插入事务的 t_xmin 为 600，tuple2 的 t_xmin 为 610，tuple3 的 t_xmin 为 620，冻结过程如下：

1. 位于 1 号块在 tuple4 的行片头插入事务的 t_xmin 为 630，tuple5 的 t_xmin 为 640。
2. 位于 2 号块在 tuple6 的行片头插入事务的 t_xmin为 650，tuple7 的 t_xmin 为 660，tuple8 的 t_xmin 为 900。

那么再次假设在 tuple1 上 UPDATE 或者 DELETE 后的 t_xmax 为 700，在 tuple6 上的 t_xmax 为 750。此刻冻结过程进行如下：

1. 第 0 号页面上的三条元组的t_xmin小于 limit 的值，因此都将被冻结，并且tuple1上面有删除标识，因此清理时将被移除，1号页面在VM中对所有元组可见，则跳过清理。
2. 2号页面的tuple6和tuple7将被冻结，tuple 7被移除。

#### 1.3.2 Aggressive模式下的冻结

Datfrozenxied是系统表  pg_database中的一列，该列中保存每个数据库最老的已冻结的事务标识。Vacuum_freeze_table_age 默认值为1.5亿。

假设现在系统当前的最早的事务 `oldestXmin` 为 12000，而 `vacuum_freeze_min_age` 的值为 10000 ，假设 `vacuum_freeze_table_age` 为 10000 冻结点的事务 id 为 2000。此刻 0 号页面的元组都会被冻结，1 并进行扫描检查。1 号页面元组冻结，在 lazy 模式下跳过该页。2 号页面，9999 将会被冻结，10010 不会被冻结。

### 1.4 事务回卷

当事务开始（执行begin第一条命令时），事务管理器会为该事务分配一个txid（transaction id）作为唯一标识符。txid是一个32位无符号整数，取值空间大小约42亿（2^32-1）。

txid可通过txid_current()[函数](https://marketing.csdn.net/p/3127db09a98e0723b83b2914d9256174?pId=2782?utm_source=glcblog&spm=1001.2101.3001.7020)获取

```sql
testdb=# BEGIN;
BEGIN
testdb=# SELECT txid_current();
 txid_current 
--------------
          100
(1 row)
```

**三个特殊的txid**

- 0：InvalidTransactionId，表示无效的事务ID
- 1：BootstrapTransactionId，表示系统表初始化时的事务ID，比任何普通的事务ID都旧。
- 2：FrozenTransactionId，冻结的事务ID，比任何普通的事务ID都旧。
- 大于2的事务ID都是普通的事务ID。

**事务间的可见性**

txid间可以相互比较大小，任何事务只可见txid＜其自身txid的事务修改结果。pg将txid空间视为一个环，若不进行特殊处理，txid到达最大值后又会从3开始分配（0-2保留），如果进行简单的比大小，之前的事务就可以看到这个新事务创建的元组，而新事务不能看到之前事务创建的元组，这违反了事务的可见性。这种现象称为PG的事务ID回卷问题。

实际上虽然txid空间有42亿，却并非按实际数字大小来判断可见性。pg将txid空间一分为二，对于某个特定的txid，其后约21亿个txid属于未来，均不可见；其前约21亿个txid属于过去，均可见。

例如对于txid=100的事务，从101到2^31+100均为可见事务（即n+1到n+2^31）；从2^31+101到99均为可见事务（即n+2^31+1到n-1）。

![Fig. 5.1. Transaction ids in PostgreSQL.](https://i-blog.csdnimg.cn/blog_migrate/c0c8bfd045b25f3520f2f2bbdfd12137.png)

代码中的实际比较方法：

```c
/* * TransactionIdPrecedes --- is id1 logically < id2? */
bool TransactionIdPrecedes(TransactionId id1, TransactionId id2) // 结果返回一个bool值
  int32		diff;
	//若其中一个不是普通id，则其一定较新（较大）
	if (!TransactionIdIsNormal(id1) || !TransactionIdIsNormal(id2))
  return (id1 < id2);
	diff = (int32) (id1 - id2);
	return (diff < 0);}
```

#### 1.4.1 比较特殊事务与普通事务txid

首先利用TransactionIdIsNormal判断当前txid是不是普通的txid（即txid>3），前面说过0-2都是保留的txid，它们比任何普通txid都要旧。

- 0：InvalidTransactionId，表示无效的事务 ID
- 1：BootstrapTransactionId，表示系统表初始化时的事务 ID，比任何普通的事务ID都旧。
- 2：FrozenTransactionId，冻结的事务 ID，比任何普通的事务ID都旧。
- 大于2的事务ID都是普通的事务ID。

比较方法非常简单，就通过

```c
if (!TransactionIdIsNormal(id1) || !TransactionIdIsNormal(id2))
return (id1 < id2);
```

可以代入值实验一下：

- 若id1=10，id2=2，return(10<2)。明显10<2为假，所以10比2大，普通事务较新。
- 若id1=2，id2=10，return(2<10)。2<10为真，所以10比2大，还是普通事务较新。

#### 1.4.2 普通事务间的比较

这里其实用到一个小技巧，把两个事务ID相减后转为int 32类型。

```c
	diff = (int32) (id1 - id2);
  return (diff < 0);
```

由于int 32带符号，需要用第一位表示符号位，所以它能表示的正数比unsigned int 32类型少一半，int 32的数据取值范围为[-2^(n-1),2^(n-1)-1]，即[-2^31,2^31-1]。当两个txid相减结果>2^31时，转为int 32后其实是个负数（符号位从0变成了1）。

我们用回前面图的例子，id1=2^31+101，id2=100。id1-id2=2^31+1，用二进制表示即：100…中间30个0…001。当转为int 32后，由于第一位为符号位，而1表示负数，所以转换后这个值其实就是-1，小于0，因此txid=2^31+101的事务反而要旧。

这样的方法是不是就不会再有问题了呢？其实不是，如果图中的100真的是非常非常旧的事务，那它确实应该被2^31+101这个事务看见，此时上面的判断就是错的。

也就是说如果id2确实是回卷前的txid，上面的判断方法就会出现问题。为了避免这种问题，pg必须保证一个数据库中两个有效的事务之间的年龄最多是2^31（同一个数据库中，存在的最旧和最新两个事务txid相差不得超过2^31）。
