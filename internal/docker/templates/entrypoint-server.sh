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

JVM_PROFILE="${JVM_PROFILE:-aikar}"
case "$JVM_PROFILE" in
    aikar)
        GC_OPTS="-XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:MaxGCPauseMillis=200 \
-XX:+UnlockExperimentalVMOptions -XX:+DisableExplicitGC -XX:+AlwaysPreTouch \
-XX:G1NewSizePercent=30 -XX:G1MaxNewSizePercent=40 -XX:G1HeapRegionSize=8M \
-XX:G1ReservePercent=20 -XX:G1HeapWastePercent=5 -XX:G1MixedGCCountTarget=4 \
-XX:InitiatingHeapOccupancyPercent=15 -XX:G1MixedGCLiveThresholdPercent=90 \
-XX:G1RSetUpdatingPauseTimePercent=5 -XX:SurvivorRatio=32 -XX:+PerfDisableSharedMem \
-XX:MaxTenuringThreshold=1 -Dusing.aikars.flags=https://mcflags.emc.gs -Daikars.new.flags=true"
        ;;
    aikar-large)
        GC_OPTS="-XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:MaxGCPauseMillis=200 \
-XX:+UnlockExperimentalVMOptions -XX:+DisableExplicitGC -XX:+AlwaysPreTouch \
-XX:G1NewSizePercent=40 -XX:G1MaxNewSizePercent=50 -XX:G1HeapRegionSize=16M \
-XX:G1ReservePercent=15 -XX:G1HeapWastePercent=5 -XX:G1MixedGCCountTarget=4 \
-XX:InitiatingHeapOccupancyPercent=20 -XX:G1MixedGCLiveThresholdPercent=90 \
-XX:G1RSetUpdatingPauseTimePercent=5 -XX:SurvivorRatio=32 -XX:+PerfDisableSharedMem \
-XX:MaxTenuringThreshold=1 -Dusing.aikars.flags=https://mcflags.emc.gs -Daikars.new.flags=true"
        ;;
    *)
        echo "ERROR: Unknown JVM_PROFILE: $JVM_PROFILE (valid: aikar, aikar-large)" >&2
        exit 1
        ;;
esac

JVM_OPTS="${MEMORY_OPTS} ${GC_OPTS} ${JAVA_OPTS:-}"

if [ -d /server/config ]; then
    find /server/config -type f | while read -r file; do
        envsubst < "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
    done
fi

find /server/plugins -type f \( -name '*.yml' -o -name '*.yaml' -o -name '*.conf' \) 2>/dev/null | while read -r file; do
    envsubst < "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
done

if [ -d /server/.embedded-world ]; then
    if [ ! "$(ls -A /server/world 2>/dev/null)" ]; then
        mkdir -p /server/world
        cp -r /server/.embedded-world/* /server/world/
    fi
fi

if [ -d /server/.embedded-plugins ]; then
    if ! ls /server/plugins/*.jar 1>/dev/null 2>&1; then
        cp -r /server/.embedded-plugins/* /server/plugins/
    fi
fi

if [ -d /extra-plugins ] && ls /extra-plugins/*.jar 1>/dev/null 2>&1; then
    echo "WARNING: extra-plugins detected, do not use in production" >&2
    cp /extra-plugins/*.jar /server/plugins/
fi

exec java ${JVM_OPTS} -jar server.jar --nogui
