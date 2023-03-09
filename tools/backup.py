#!/usr/local/bin/env python3
# -*- coding: utf-8 -*-

# 依赖 pymysql dsnparse ops_channel
# pip install pymysql dsnparse ops_channel


from ops_channel import cli

dsn='mysql://root:mock@localhost:63307/mock'
#备份到指定目录
dir='./'

conn = cli.get_mysql_connection(dsn)
tables = cli.get_mysql_tables(conn)
#可以移除不需要备份的表
#tables.remove('audit_log_tab')

#备份表结构
cli.dump_mysql_ddl(conn, dir=dir)

#备份数据
cli.dump_mysql_data(conn, dir=dir, limit=10000, overwrite=True, tables=tables)

