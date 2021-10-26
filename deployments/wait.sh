#!/bin/bash
# Ждет успешного завершения контейнера и выходит со $status контейнера в случае ошибки
status=$(docker wait $1)
exit $status
