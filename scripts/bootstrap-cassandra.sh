#!/bin/sh
set -eu

kubectl create configmap cassandra-schema \
  --from-file=db.cql=./db.cql \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -k k8s/cassandra
