---
title: "Python 漏洞分析"
lastmod: "2024-11-26T11:51:19+08:00"
---

[TOC]

## Python 输入验证错误漏洞（CVE-2023-24329）

Python 3.11之前版本存在安全漏洞，需要升级到 3.11 及以上版本。

**漏洞描述**：https://avd.aliyun.com/detail?id=AVD-2023-24329&timestamp__1384=euD%3DDK7IjDkGkzD%2FtR2dxmoqfxmq5RurwpD

**解决方法**：

1. 升级 Python 到 3.12
2. 升级 Python 3.9.17 及以上版本（版本描述：https://docs.python.org/release/3.8.20/whatsnew/changelog.html#python-3-8-17-final）

## Python 安全漏洞（CVE-2023-36632）

Python 3.11.4 之前的 email.utils.parseaddr 函数允许攻击者通过精心制作的参数触发“RecursionError：在调用 Python 对象时超出最大递归深度”。此参数可能是应用程序输入数据中的不受信任值，该数据原本应包含名称和电子邮件地址。注意：在 Python 电子邮件包的文档中，email.utils.parseaddr 被归类为遗留 API。应用程序应改用 email.parser.BytesParser 或 email.parser.Parser 类。

**漏洞描述**：https://avd.aliyun.com/detail?id=AVD-2023-36632&timestamp__1384=eqAx0iG%3DGQeWqYvxGNuCDUxQu8PTor8tH4D

**解决方法**：

暂无，该问题的[github描述](https://github.com/github/advisory-database/pull/2722)地址。

![](https://pic.imgdb.cn/item/67453e8cd0e0a243d4d0d957.png)

RedHat 漏洞记录：https://access.redhat.com/security/cve/cve-2023-36632，未提供修复更新。

![](https://pic.imgdb.cn/item/6745453cd0e0a243d4d0e70c.png)

## Python 信息泄露漏洞（CVE-2023-40217）

Python 3.8.18 之前、3.9.x 版本在 3.9.18 之前、3.10.x 版本在 3.10.13 之前以及 3.11.x 版本在 3.11.5 之前发现了一个问题。它主要影响使用 TLS 客户端身份验证的服务器（如 HTTP 服务器）。如果在创建 TLS 服务器端套接字、将数据接收进套接字缓冲区后迅速关闭套接字，那么在 SSLSocket 实例检测套接字为“未连接”且不会启动握手的过程中，缓冲区中的数据仍然可读。如果服务器端 TLS 对端期望客户端证书身份验证，则这些数据将不会被验证，并且与有效的 TLS 流数据无法区分。数据大小限制在缓冲区能容纳的范围内。（由于易受攻击的代码路径需要在 SSLSocket 初始化时关闭连接，因此 TLS 连接不能直接用于数据泄露。）

**漏洞描述**：https://avd.aliyun.com/detail?id=AVD-2023-40217&timestamp__1384=eqjxcDRDnQQxgDBqDTCDUxQq4OzXxmdDCWjeD

**解决方法**：

1. 升级 Python 3.12
2. 升级 python 3.9.18 及以上版本（版本描述：https://docs.python.org/release/3.8.20/whatsnew/changelog.html#python-3-8-18-final）

## Python 安全漏洞(CVE-2023-27043)

Python 2.7.18之前版本、3.x版本至3.11版本存在安全漏洞，该漏洞源于电子邮件模块错误地解析包含特殊字符的电子邮件地址。

**漏洞描述**：https://avd.aliyun.com/detail?id=AVD-2023-27043&timestamp__1384=euD%3D0I4IOtD50%3DD%2FD0Y5iPiK%3DAKits1W%2BqH4D

**解决方法**：

1. 升级 Python 3.12
2. 升级 Python 3.9.20（版本描述：https://docs.python.org/release/3.8.20/whatsnew/changelog.html#python-3-8-20-final）

## 总结

除了 [CVE-2023-36632](#cve-2023-36632) 漏洞暂无修复方法，其余的漏洞可以通过升级 Python 版本解决，考虑到升级 Python 3.10 以上的版本需要保证 openssl 版本大于 1.1.1，所以建议把 Python 版本升级到 **3.9.20** 以解决上述漏洞问题。
