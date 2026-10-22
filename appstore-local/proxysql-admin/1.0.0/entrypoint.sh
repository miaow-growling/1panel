#!/bin/sh
# 从环境变量渲染 proxysql 配置后启动
sed -e "s/__ADMIN_PWD__/$PSQL_ADMIN_PASSWORD/g" \
    -e "s/__MONITOR_PWD__/$MONITOR_PASSWORD/g" \
    -e "s/__BACKEND_HOST__/$MYSQL_BACKEND_HOST/g" \
    -e "s/__BACKEND_PORT__/$MYSQL_BACKEND_PORT/g" \
    /tpl/proxysql.cnf.tpl > /etc/proxysql.cnf
exec proxysql -f -c /etc/proxysql.cnf --initial
