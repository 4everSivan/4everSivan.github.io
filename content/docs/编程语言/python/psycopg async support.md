---
title: "Psycopg2 Asynchronous support"
---

# Psycopg2 Asynchronous support

Psycopg可以向PostgreSQL数据库发出异步查询。通过将参数 `async=1` 传递给connect()函数来建立异步通信样式:返回的连接将在异步模式下工作。

在异步模式下，Psycopg连接将依赖于调用者轮询套接字文件描述符，检查它是否准备好接受数据，或者查询结果是否已经传输并准备好在客户机上读取。调用者可以使用方法 `fileno()` 来获取连接文件描述符和poll()来根据当前连接状态进行通信。

下面是一个使用 `fileno()` 和 `poll()` 方法以及Python `select()` 函数的循环示例，以便对Psycopg进行异步操作:

```python
def wait(conn):
    while True:
        state = conn.poll()
        if state == psycopg2.extensions.POLL_OK:
            break
        elif state == psycopg2.extensions.POLL_WRITE:
            select.select([], [conn.fileno()], [])
        elif state == psycopg2.extensions.POLL_READ:
            select.select([conn.fileno()], [], [])
        else:
            raise psycopg2.OperationalError("poll() returned %s" % state)
```

上面的循环当然会阻塞整个应用程序:在真正的异步框架中，会在许多文件描述符上调用 `select()`，等待其中任何一个文件描述符准备就绪。尽管如此，该函数只能使用非阻塞命令连接到 PostgreSQL 服务器，并且获得的连接可以用于执行进一步的非阻塞查询。在 `poll()` 返回 `POLL_OK` 之后，也就是 `wait()` 返回之后，连接就可以安全使用了:

```python
>>> aconn = psycopg2.connect(database='test', async=1)
>>> wait(aconn)
>>> acurs = aconn.cursor()
```

注意，为了实现完全非阻塞的连接尝试，还需要满足一些其他要求:请参阅PQconnectStart()的libpq文档。

同样的循环也应该用于执行非阻塞查询:在通过 `execute()` 或 `callproc()` 发送查询之后，在游标可用的连接上调用 `poll()`。连接，直到返回 `POLL_OK`，此时查询已经完全发送到服务器，如果它产生了数据，结果已经传输到客户端，可以使用常规游标方法:

```python
>>> acurs.execute("SELECT pg_sleep(5); SELECT 42;")
>>> wait(acurs.connection)
>>> acurs.fetchone()[0]
42
```

当执行异步查询时，`connection. isexecution()` 返回 `True`。两个游标不能在同一个异步连接上执行并发查询。

使用异步连接有几个限制:连接总是处于自动提交模式，并且不可能更改它。因此，事务不会在第一次查询时隐式启动，也不可能使用 `commit()` 和 `rollback()` 方法:您可以使用 `execute()` 手动控制事务，以发送诸如` BEGIN`、`commit` 和 `rollback` 等数据库命令。类似地，不能使用 `set_session()`，但仍然可以使用适当的`default_transaction_`…参数。

对于异步连接，也不可能使用 `set_client_encoding()`、`executemany()`、大对象、命名游标。

COPY命令在异步模式下也不受支持，但这可能在未来的版本中实现。



## Connection method

### 1. poll()

> `poll()`
>
> Used during an asynchronous connection attempt, or when a cursor is executing a query on an asynchronous connection, make communication proceed if it wouldn’t block.
>
> Return one of the constants defined in [Poll constants](https://www.psycopg.org/docs/extensions.html#poll-constants). If it returns [`POLL_OK`](https://www.psycopg.org/docs/extensions.html#psycopg2.extensions.POLL_OK) then the connection has been established or the query results are available on the client. Otherwise wait until the file descriptor returned by [`fileno()`](https://www.psycopg.org/docs/connection.html#connection.fileno) is ready to read or to write, as explained in [Asynchronous support](https://www.psycopg.org/docs/advanced.html#async-support). [`poll()`](https://www.psycopg.org/docs/connection.html#connection.poll) should be also used by the function installed by [`set_wait_callback()`](https://www.psycopg.org/docs/extensions.html#psycopg2.extensions.set_wait_callback) as explained in [Support for coroutine libraries](https://www.psycopg.org/docs/advanced.html#green-support).
>
> [`poll()`](https://www.psycopg.org/docs/connection.html#connection.poll) is also used to receive asynchronous notifications from the database: see [Asynchronous notifications](https://www.psycopg.org/docs/advanced.html#async-notify) from further details.

`poll ()` : 在异步连接尝试期间使用，或者当游标在异步连接上执行查询时使用，如果通信不阻塞，则通信继续进行。

返回Poll常量中定义的一个常量。如果它返回POLL_OK，那么连接已经建立，或者查询结果在客户机上可用。否则等待，直到 `fileno()` 返回的文件描述符准备好了可以读或写，如异步支持中所述。poll()也应该由 `set_wait_callback()` 安装的函数使用，详见对协程库的支持。

poll()还用于从数据库接收异步通知:请参阅来自更多详细信息的异步通知。

### 2. fileno()

> `fileno()`
>
> Return the file descriptor underlying the connection: useful to read its status during asynchronous communication.

`fileno()` : 返回连接下面的文件描述符:在异步通信期间读取其状态很有用。
