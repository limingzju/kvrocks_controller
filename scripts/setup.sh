#!/usr/bin/env bash
cd docker && docker-compose -p kvrocks-controller up -d --force-recreate && cd ..