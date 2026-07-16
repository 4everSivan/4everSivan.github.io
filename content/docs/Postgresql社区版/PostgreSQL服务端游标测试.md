---
title: "PostgreSQL服务端游标测试"
---

## 测试环境

```sql
PG版本： 
	PostgreSQL 14.2

测试表结构：
	create table tb_test3( id int,name name );
```

## 测试内容

```
测试游标创建后，能否获取后续插入的新的数据。
```

## 测试步骤

```sql
-- 会话1中定义游标
declare test_select CURSOR with hold for select * from tb_test3 ;

-- 创建新会话会话2，在会话2中插入数据
postgres=# insert into tb_test3 values (100,'aa');
INSERT 0 1
postgres=# insert into tb_test3 values (101,'ab');
INSERT 0 1
postgres=# insert into tb_test3 values (103,'ac');
INSERT 0 1
postgres=# insert into tb_test3 values (104,'ad');
INSERT 0 1

-- 会话1 查询数据
postgres=# select * from tb_test3 order by id desc;
 id  | name  
-----+-------
 104 | ad
 103 | ac
 101 | ab
 100 | aa
  31 | sivan
--More--

-- 会话1 执行游标
postgres=# fetch 10 test_select ;
 id | name  
----+-------
  1 | sivan
  2 | sivan
  3 | sivan
  4 | sivan
  5 | sivan
  6 | sivan
  7 | sivan
  8 | sivan
  9 | sivan
 10 | sivan
(10 rows)

postgres=# fetch 10 test_select ;
 id | name  
----+-------
 11 | sivan
 12 | sivan
 13 | sivan
 14 | sivan
 15 | sivan
 16 | sivan
 17 | sivan
 18 | sivan
 19 | sivan
 20 | sivan
(10 rows)

postgres=# fetch 10 test_select ;
 id | name  
----+-------
 21 | sivan
 22 | sivan
 23 | sivan
 24 | sivan
 25 | sivan
 26 | sivan
 27 | sivan
 28 | sivan
 29 | sivan
 30 | sivan
(10 rows)

postgres=# fetch 10 test_select ;
 id | name  
----+-------
 31 | sivan
(1 row)

-- 再次执行已经获取不到数据了

postgres=# fetch 10 test_select ;
 id | name 
----+------
(0 rows)

```

## 测试结果

```
在定义游标后插入的数据无法通过游标获取，游标只能获取到定义游标之前数据库中的数据
```

