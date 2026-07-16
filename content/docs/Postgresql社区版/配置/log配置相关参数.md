---
title: "log配置相关参数"
---

log_destination = 'csvlog'
logging_collector = on
log_directory = 'pg_log'
log_filename = 'postgresql-%Y-%m-%d_%H%M%S.log'
log_rotation_age = 1d
log_rotation_size = 100MB
log_min_messages = info
\# 记录执行慢的SQL
log_min_duration_statement = 60
log_checkpoints = on
log_connections = on
log_disconnections = on
log_duration = on
log_line_prefix = '%m'
\# 监控数据库中长时间的锁
log_lock_waits = on
\# 记录DDL操作
log_statement = 'ddl'

#log_rotation_age = 1d            # Automatic rotation of logfiles will   # 单个日志的生存期，默认为1天，在日志文件没有达到log_rotation_size
                    \# happen after that time. 0 disables.        # 时，一天只生成一个日志文件
\#log_rotation_size = 10MB        # Automatic rotation of logfiles will   # 单个日志文件大小，如果时间没有超过log_rotation_age，一个日志
                    \# happen after that much log output.         # 文件最大只能是10M，否则生成一个新的日志文件

#log_min_messages = warning        # values in order of decreasing detail: # 控制写到服务器日志里的信息的详细程度。有效值是DEBUG5， DEBUG4， 
                    \#  debug5                      # DEBUG3，DEBUG2，DEBUG1， INFO，NOTICE，WARNING， ERROR，LOG
                    \#  debug4                      # FATAL， and PANIC。每个级别都包含它后面的级别。越靠后的数值 
                    \#  debug3                      # 发往服务器日志的信息越少，缺省是WARNING。
                    \#  debug2
                    \#  debug1
                    \#  info
                    \#  notice
                    \#  warning
                    \#  error
                    \#  log
                    \#  fatal
                    \#  panic

\#log_min_error_statement = error    # values in order of decreasing detail:
                    \#  debug5                      # 控制是否在服务器日志里输出那些导致错误条件的 SQL 语句。
                    \#  debug4                      # 所有导致一个特定级别(或者更高级别)错误的 SQL 语句都要
                    \#  debug3                      # 被记录。有效的值有DEBUG5， DEBUG4，DEBUG3， 
                    \#  debug2                      # DEBUG2，DEBUG1，INFO，NOTICE，WARNING，ERROR，LOG，FATAL
                    \#  debug1                      # ，和PANIC。缺省是ERROR，表示所有导致错误、致命错误、恐慌的
                    \#  info                       # SQL语句都将被记录。
                    \#  notice
                    \#  warning
                    \#  error
                    \#  log
                    \#  fatal
                    \#  panic (effectively off)

log_min_duration_statement = 0    # -1 is disabled, 0 logs all statements # 这个参数非常重要，是排查慢查询的好工具，-1是关闭记录这类日志
                    \# and their durations, > 0 logs only         # 0 是记录所有的查询SQL，如果设置为大于0（毫秒），则超过该值的
                    \# statements running at least this number      # 执行时间的sql会记录下来
                    \# of milliseconds
