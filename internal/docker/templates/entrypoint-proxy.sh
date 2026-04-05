#!/bin/sh
set -e

if [ -n "$JAVA_MEMORY" ]; then
    MEMORY_OPTS="-Xms${JAVA_MEMORY} -Xmx${JAVA_MEMORY}"
elif [ -n "$JVM_RAM_PERCENTAGE" ]; then
    MEMORY_OPTS="-XX:InitialRAMPercentage=${JVM_RAM_PERCENTAGE} -XX:MaxRAMPercentage=${JVM_RAM_PERCENTAGE}"
else
    echo "ERROR: Either JAVA_MEMORY or JVM_RAM_PERCENTAGE must be set" >&2
    exit 1
fi

GC_OPTS="-XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:+UnlockExperimentalVMOptions \
-XX:+AlwaysPreTouch -XX:G1HeapRegionSize=4M -XX:MaxInlineLevel=15"

JVM_OPTS="${MEMORY_OPTS} ${GC_OPTS} ${JAVA_OPTS:-}"

if [ -n "$FORWARDING_SECRET" ]; then
    echo "$FORWARDING_SECRET" > /server/forwarding.secret
fi

find /server -maxdepth 1 -type f \( -name '*.toml' -o -name '*.yml' -o -name '*.yaml' -o -name '*.conf' -o -name '*.txt' \) 2>/dev/null | while read -r file; do
    envsubst < "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
done

find /server/plugins -type f \( -name '*.yml' -o -name '*.yaml' -o -name '*.conf' \) 2>/dev/null | while read -r file; do
    envsubst < "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
done

if ls /server/.embedded-plugins/*.jar 1>/dev/null 2>&1; then
    if ! ls /server/plugins/*.jar 1>/dev/null 2>&1; then
        cp -r /server/.embedded-plugins/* /server/plugins/
    fi
fi

if [ -d /extra-plugins ] && ls /extra-plugins/*.jar 1>/dev/null 2>&1; then
    echo "WARNING: extra-plugins detected, do not use in production" >&2
    cp /extra-plugins/*.jar /server/plugins/
fi

exec java ${JVM_OPTS} -jar server.jar
