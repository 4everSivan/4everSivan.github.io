---
title: "PostgreSQL prepare"
---

# PostgreSQL prepare

prepare的使用：

在执行一个SQL时，首先生成执行计划（进行语义分析、词法解析、逻辑优化、物理优化）、执行、结果传输等操作。
如果一个SQL在应用中反复使用，我们可以将此SQL参数化，只做一次prepare，后面执行时就不需要进行前面执行计划的生成操作，直接使用prepare好的执行计划。

对于比较长的SQL、参数较固定的SQL，可以使用prepare，下面做个简单的举例：

**特点：**

- Prepared语句只在`session`的整个生命周期中存在，一旦`session`结束，Prepared语句也不存在了。如果下次再使用需重新创建。
- Prepared语句不能在多个并发的 client 中共有。
- prepared语句可以通过DEALLOCATE命令清除。
- 当前`session`的`prepared`语句：`pg_prepared_statements`

**使用：**

1. 存储过程

```sql
DO
$$
DECLARE
ret_ref refcursor;
one_row record;
BEGIN
PREPARE test_pre(int, text) AS INSERT INTO test values($1, $2);
--EXECUTE test_pre(1, 'test_pre'); --如果不用execute包一层，会认为是个函数，会报错，要用下面的
EXECUTE 'EXECUTE test_pre(1, ''test_pre'')';
DEALLOCATE PREPARE test_pre;
 
OPEN ret_ref FOR SELECT * FROM test;
 
FETCH ret_ref INTO one_row;
WHILE ret_ref%FOUND LOOP
raise notice 'id is: %, text is: %', one_row.id, one_row.text;
FETCH NEXT IN ret_ref INTO one_row;
END LOOP;
 
CLOSE ret_ref;
--truncate test;
END
$$
```

2. 在客户端上使用：

   可以在管理工具、命令行工具直接使用，session级别生效：

```sql
PREPARE test_pre1(int, text) AS INSERT INTO test values($1, $2);
EXECUTE test_pre1(1, 'test_pre');
 
SELECT * FROM test;
```

