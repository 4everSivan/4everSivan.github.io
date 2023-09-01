(window.webpackJsonp=window.webpackJsonp||[]).push([[28],{311:function(s,t,a){"use strict";a.r(t);var e=a(14),r=Object(e.a)({},(function(){var s=this,t=s._self._c;return t("ContentSlotsDistributor",{attrs:{"slot-key":s.$parent.slotKey}},[t("h1",{attrs:{id:"postgresql数据库在linux上的安装搭建-单例"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#postgresql数据库在linux上的安装搭建-单例"}},[s._v("#")]),s._v(" PostgreSQL数据库在Linux上的安装搭建（单例）")]),s._v(" "),t("h2",{attrs:{id:"_1-环境信息"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_1-环境信息"}},[s._v("#")]),s._v(" 1. 环境信息")]),s._v(" "),t("div",{staticClass:"language-shell extra-class"},[t("pre",{pre:!0,attrs:{class:"language-shell"}},[t("code",[t("span",{pre:!0,attrs:{class:"token number"}},[s._v("1")]),s._v(". VMware Workstation Pro\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("2")]),s._v(". CentOS-7-x86_64-Everything-2009\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("3")]),s._v(". PostgreSQl "),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("14.1")]),s._v("\n")])])]),t("h2",{attrs:{id:"_2-问题"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_2-问题"}},[s._v("#")]),s._v(" 2. 问题")]),s._v(" "),t("div",{staticClass:"language- extra-class"},[t("pre",{pre:!0,attrs:{class:"language-text"}},[t("code",[s._v("在 VMware 虚拟机中安装 PostageSQL 数据库（单例）\n")])])]),t("h2",{attrs:{id:"_3-操作步骤"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_3-操作步骤"}},[s._v("#")]),s._v(" 3. 操作步骤")]),s._v(" "),t("h3",{attrs:{id:"_3-1-虚拟机配置"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_3-1-虚拟机配置"}},[s._v("#")]),s._v(" 3.1 虚拟机配置")]),s._v(" "),t("div",{staticClass:"language-shell extra-class"},[t("pre",{pre:!0,attrs:{class:"language-shell"}},[t("code",[t("span",{pre:!0,attrs:{class:"token number"}},[s._v("1")]),s._v(". 配置 VMware 虚拟机为NAT网络模式,共享物理机网络\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("2")]),s._v(". 配置虚拟机的网络状态\n\t"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("vi")]),s._v(" /etc/sysconfig/network-scripts/ifcfg-ens33 \n\t\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("3")]),s._v(". 修改配置如下（配置完需要重启网络服务）\n\t"),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("TYPE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("Ethernet\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PROXY_METHOD")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("none\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("BROWSER_ONLY")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("no\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("BOOTPROTO")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("static\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("DEFROUTE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("yes\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("IPADDR")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("192.168")]),s._v(".6.177\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("NETMASK")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("255.255")]),s._v(".255.0\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("NAME")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("ens33\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("DNS1")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("223.5")]),s._v(".5.5\t\t \t\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("#阿里yum源")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("DNS2")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("8.8")]),s._v(".8.8\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("GATEWAY")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("192.168")]),s._v(".6.2  \t\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("#NAT模式下的虚拟网卡网关")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("UUID")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("96765599")]),s._v("-1f43-4189-b998-b3f1ace198d1\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("DEVICE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("ens33\n    "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("ONBOOT")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("yes\n    \n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("4")]),s._v(". 下载Cenos-7.repo到/etc/yum/repos.d/，并更名为CentOS-Base.repo\n\t"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("curl")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-o")]),s._v(" /etc/yum.repos.d/CentOS-Base.repo https://mirrors.aliyun.com/repo/Centos-7.repo\n\t\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("5")]),s._v(". 本地缓存阿里源上的软件包信息\n\tyum makecache\n\t"),t("span",{pre:!0,attrs:{class:"token punctuation"}},[s._v("(")]),s._v("注意：如果出现 "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("timeout")]),s._v(" 报错，检查一下配置静态IP时，有没有配置DNS服务器，以及网关的配置"),t("span",{pre:!0,attrs:{class:"token punctuation"}},[s._v(")")]),s._v("\n")])])]),t("h3",{attrs:{id:"_3-2-添加数据库用户"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_3-2-添加数据库用户"}},[s._v("#")]),s._v(" 3.2 添加数据库用户")]),s._v(" "),t("div",{staticClass:"language-shell extra-class"},[t("pre",{pre:!0,attrs:{class:"language-shell"}},[t("code",[t("span",{pre:!0,attrs:{class:"token function"}},[s._v("groupadd")]),s._v(" postgres\n"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("useradd")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-g")]),s._v(" postgres postgres\n"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("passwd")]),s._v(" postgres\n")])])]),t("h3",{attrs:{id:"_3-3-安装依赖"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_3-3-安装依赖"}},[s._v("#")]),s._v(" 3.3 安装依赖")]),s._v(" "),t("div",{staticClass:"language-shell extra-class"},[t("pre",{pre:!0,attrs:{class:"language-shell"}},[t("code",[s._v("yum "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-y")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("install")]),s._v(" lrzsz sysstat e4fsprogs ntp readline-devel zlib zlib-devel openssl openssl-devel pam-devel libxml2-devel libxslt-devel python-devel tcl-devel gcc "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("make")]),s._v(" flex bison perl perl-devel perl-ExtUtils* OpenIPMI-tools systemtap-sdt-devel smartmontools libcurl "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("vim")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("wget")]),s._v(" systemd-devel\n")])])]),t("h3",{attrs:{id:"_3-4-安装及修改配置文件"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_3-4-安装及修改配置文件"}},[s._v("#")]),s._v(" 3.4 安装及修改配置文件")]),s._v(" "),t("div",{staticClass:"language-shell extra-class"},[t("pre",{pre:!0,attrs:{class:"language-shell"}},[t("code",[t("span",{pre:!0,attrs:{class:"token number"}},[s._v("1")]),s._v(". 下载解压PostgreSQL\n\t"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("mkdir")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-p")]),s._v(" /opt/soft\n\t"),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("cd")]),s._v(" /opt/soft\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("wget")]),s._v(" https://ftp.postgresql.org/pub/source/v14.1/postgresql-14.1.tar.gz --no-check-certificate\t\t\t\t\t\t\t\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("#跳过证书认证")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("tar")]),s._v(" zxvf postgresql-14.1.tar.gz\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("2")]),s._v(". 源码安装\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("cd")]),s._v(" /opt/soft/postgresql-14.1\n    ./configure "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("--prefix")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("/usr/local/pgsql/14.1\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# --prefix 指定安装路径")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("make")]),s._v(" world "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-j32")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("&&")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("make")]),s._v(" install-world "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-j32")]),s._v("\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# make world 构建所有内容")]),s._v("\n    \t\t\t\t\t\t\t\t\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# make install-world 安装数据库和文档")]),s._v("\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("3")]),s._v(". 开启调试\n\t./configure "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("--prefix")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("/usr/local/pgsql/14.1/ --with-pgport"),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("5432")]),s._v(" --enable-dtrace --enable-debug\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# --with-pgport 调整默认端口\t--enable-dtrace 允许动态跟踪工具 DTrace\t")]),s._v("\n\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("#--enable-debug 把所有程序和库以带有调试符号的方式编译")]),s._v("\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("2")]),s._v(". 创建数据库目录及权限配置\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("mkdir")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-p")]),s._v(" /data/pgsql/14.1/pgdata\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("chown")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-R")]),s._v(" postgres:postgres /data/pgsql\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("chown")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-R")]),s._v(" postgres:postgres /usr/local/pgsql/14.1\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("chmod")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-R")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("775")]),s._v(" /usr/local/pgsql/14.1\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# 775满权限")]),s._v("\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("4")]),s._v(". 添加局部环境变量"),t("span",{pre:!0,attrs:{class:"token punctuation"}},[s._v("(")]),s._v("/home/postgres/.bashrc"),t("span",{pre:!0,attrs:{class:"token punctuation"}},[s._v(")")]),s._v("\n\t"),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PGHOME")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("/usr/local/pgsql/14.1\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PGPORT")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("5432")]),s._v("\t\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# 默认端口5432（记得修改postgresql.conf文件）")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PGDATA")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("/data/pgsql/14.1/pgdata\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("DATE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token variable"}},[t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("`")]),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("date")]),s._v(" +"),t("span",{pre:!0,attrs:{class:"token string"}},[s._v('"%Y%m%d%H%M"')]),t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("`")])]),s._v("\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# 日期格式")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PGUSER")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("postgres\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PGHOST")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("localhost\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PGDATABASE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("postgres\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[t("span",{pre:!0,attrs:{class:"token environment constant"}},[s._v("PATH")])]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("$PGHOME")]),s._v("/bin:"),t("span",{pre:!0,attrs:{class:"token environment constant"}},[s._v("$PATH")]),s._v(":.\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("MANPATH")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("$PGHOME")]),s._v("/share/man:"),t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("$MANPATH")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("LD_LIBRARY_PATH")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("$PGHOME")]),s._v("/lib:"),t("span",{pre:!0,attrs:{class:"token variable"}},[s._v("$LD_LIBRARY_PATH")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[t("span",{pre:!0,attrs:{class:"token environment constant"}},[s._v("LANG")])]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("en_US.utf8\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PG_OOM_ADJUST_FILE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("/proc/self/oom_score_adj \t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# 指定OOM调整文件")]),s._v("\n    "),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("export")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token assign-left variable"}},[s._v("PG_OOM_ADJUST_VALUE")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("0")]),s._v("\t\t\t\t\t\t\t"),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# 指定OOM分数值")]),s._v("\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("5")]),s._v(". 修改完后刷新环境变量\n\t"),t("span",{pre:!0,attrs:{class:"token builtin class-name"}},[s._v("source")]),s._v(" /home/postgres/.bashrc\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("6")]),s._v(". 初始化\n    "),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("su")]),s._v(" - postgres\n    initdb "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-D")]),s._v(" /data/pgsql/14.1/pgdata "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-E")]),s._v(" UTF8 "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("--locale")]),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("C "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-U")]),s._v(" postgres "),t("span",{pre:!0,attrs:{class:"token comment"}},[s._v("# 文件目录 编码 区域 用户")]),s._v("\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("7")]),s._v(".启动\n\tpg_ctl "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-D")]),s._v(" /data/pgsql/14.1/pgdata "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("-l")]),s._v(" logfile start\n\n")])])]),t("h2",{attrs:{id:"_4-使用图形化工具连接数据库"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_4-使用图形化工具连接数据库"}},[s._v("#")]),s._v(" 4. 使用图形化工具连接数据库")]),s._v(" "),t("h3",{attrs:{id:"_4-1-修改配置文件"}},[t("a",{staticClass:"header-anchor",attrs:{href:"#_4-1-修改配置文件"}},[s._v("#")]),s._v(" 4.1 修改配置文件")]),s._v(" "),t("div",{staticClass:"language-shell extra-class"},[t("pre",{pre:!0,attrs:{class:"language-shell"}},[t("code",[t("span",{pre:!0,attrs:{class:"token number"}},[s._v("1")]),s._v(". 修改pg_hba.conf文件\n\t"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("vi")]),s._v(" /data/pgsql/14.1/pgdata/pg_hba.conf\n\t添加一条路由记录\n\t"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("host")]),s._v("\tall\t\tall\t\t"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("0.0")]),s._v(".0.0/0\tmd5\n\t\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("2")]),s._v(". 修改postgresql.conf文件\n\t"),t("span",{pre:!0,attrs:{class:"token function"}},[s._v("vi")]),s._v(" /data/pgsql/14.1/pgdata/postgresql.conf\n\tlistenaddress "),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v(" "),t("span",{pre:!0,attrs:{class:"token string"}},[s._v("'*'")]),s._v("\n\t\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("3")]),s._v(". 重启数据库服务\n\tpg_ctl restart\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("3")]),s._v(". 本地登录psql修改密码\n\t/password\n\n"),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("4")]),s._v(". 防火墙放行5432端口\n\tfirewall-cmd "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("--permanent")]),s._v(" --add-service"),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),s._v("http\n\tfirewall-cmd --add-port"),t("span",{pre:!0,attrs:{class:"token operator"}},[s._v("=")]),t("span",{pre:!0,attrs:{class:"token number"}},[s._v("5432")]),s._v("/tcp "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("--permanent")]),s._v("\n\tfirewall-cmd "),t("span",{pre:!0,attrs:{class:"token parameter variable"}},[s._v("--reload")]),s._v("\n")])])])])}),[],!1,null,null,null);t.default=r.exports}}]);