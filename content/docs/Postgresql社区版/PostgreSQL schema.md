---
title: "PostgreSQL Schema"
lastmod: "2026-03-03T17:10:25+08:00"
---

一个数据库包含一个或多个命名的模式，模式又包含表。模式还包含其它命名的对象，包括数据类型、函数，以及操作符。同一个对象名可以在不同的模式里使用而不会导致冲突； 比如，`schema1` 和 `myschema`都可以包含叫做`mytable` 的表。和数据库不同，模式不是严格分离的：一个用户可以访问他所连接的数据库中的任意模式中的对象，只要他有权限。

我们需要模式有以下几个主要原因：

1. 允许多个用户使用一个数据库而不会干扰其它用户。
2. 把数据库对象组织成逻辑组，让它们更便于管理。
3. 第三方的应用可以放在不同的模式中，这样它们就不会和其它对象的名字冲突。

## 1. 创建模式：

```sql
CREATE SCHEMA myschema;
```

通过以上命令可以创建名字为 `myschema` 的模式，在该模式被创建后，其便可拥有自己的一组逻辑对象，如表、视图和函数等。

## 2. public模式：

在介绍后面的内容之前，这里我们需要先解释一下public模式。每当我们创建一个新的数据库时，PostgreSQL都会为我们自动创建该模式。当登录到该数据库时，如果没有特殊的指定，我们将以该模式(public)的形式操作各种数据对象，如：

```sql
CREATE TABLE products ( ... ) 等同于 CREATE TABLE public.products ( ... )
```

## 3. 权限：

​    缺省时，用户看不到模式中不属于他们所有的对象。为了让他们看得见，模式的所有者需要在模式上赋予USAGE权限。为了让用户使用模式中的对象，我们可能需要赋予额外的权限，只要是适合该对象的。PostgreSQL根据不同的对象提供了不同的权限类型，如：

```sql
GRANT ALL ON SCHEMA myschema TO public; 
```

上面的ALL关键字将包含CREATE和USAGE两种权限。如果public模式拥有了 `myschema` 模式的CREATE权限，那么登录到该模式的用户将可以在 `myschema` 模式中创建任意对象，如：

```sql
CREATE TABLE myschema.products (
        product_no integer,
        name text,
        price numeric CHECK (price > 0),
    );
```

在为模式下的所有表赋予权限时，需要将权限拆分为各种不同的表操作，如：

```sql
ALTER DEFAULT PRIVILEGES IN SCHEMA myschema
GRANT INSERT, SELECT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER ON TABLES TO public;
```

在为模式下的所有Sequence序列对象赋予权限时，需要将权限拆分为各种不同的Sequence操作，如：

```sql
ALTER DEFAULT PRIVILEGES IN SCHEMA myschema
GRANT SELECT, UPDATE, USAGE ON SEQUENCES TO public;
```

在为模式下的所有函数赋予权限时，仅考虑执行权限，如：

```sql
ALTER DEFAULT PRIVILEGES IN SCHEMA myschema
GRANT EXECUTE ON FUNCTIONS TO public;
```

可以看出，通过以上方式在 `public` 模式下为 `myschema` 模式创建各种对象是极为不方便的。下面我们将要介绍另外一种方式，即通过role对象，直接登录并关联到 `myschema` 对象，之后便可以在 `myschema` 模式下直接创建各种所需的对象了。

```sql
--创建了和该模式关联的角色对象。
CREATE ROLE myschema LOGIN PASSWORD '123456';
--将该模式关联到指定的角色，模式名和角色名可以不相等。
CREATE SCHEMA myschema AUTHORIZATION myschema;
```

 在Linux Shell下，以 `myschema` 的角色登录到数据库 `MyTest`，在密码输入正确后将成功登录到该数据库。

```sql
/> psql -d MyTest -U myschema
    Password:
    MyTest=> CREATE TABLE test(i integer);
    CREATE TABLE
    --查看该模式下，以及该模式有权限看到的tables信息列表。
    MyTest=> \d
              List of relations
     Schema     |   Name   | Type  |  Owner
    ------------+---------+------+----------
     myschema |   test     | table  | myschema
    (1 rows)
```

## 4. 删除模式：

```sql
DROP SCHEMA myschema;
```

如果要删除模式及其所有对象，请使用级联删除：

```sql
DROP SCHEMA myschema CASCADE;
```

## 5. 模式搜索路径：

我们在使用一个数据库对象时可以使用它的全称来定位对象，然而这样做往往也是非常繁琐的，每次都不得不键入`owner_name.object_name`。PostgreSQL中提供了模式搜索路径，这有些类似于Linux中的`$PATH`环境变量，当我们执行一个Shell命令时，只有该命令位于$PATH的目录列表中，我们才可以通过命令名直接执行，否则就需要输入它的全路径名。PostgreSQL同样也通过查找一个搜索路径来判断一个表究竟是哪个表，这个路径是一个需要查找的模式列表。在搜索路径里找到的第一个表将被当作选定的表。如果在搜索路径中 没有匹配表，那么就报告一个错误，即使匹配表的名字在数据库其它的模式中存在也如此。

在搜索路径中的第一个模式叫做当前模式。除了是搜索的第一个模式之外，它还是在`CREATE TABLE`没有声明模式名的时候，新建表所属于的模式。要显示当前搜索路径，使用下面的命令：

```sql
 MyTest=> SHOW search_path;
      search_path
    ----------------
     "$user",public
    (1 row)
```

可以将新模式加入到搜索路径中，如：

```sql
SET search_path TO myschema,public;
```

为搜索路径设置指定的模式，如：

```sql
--当前搜索路径中将只是包含myschema一种模式。
SET search_path TO myschema;
```

