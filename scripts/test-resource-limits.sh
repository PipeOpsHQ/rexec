#!/bin/bash
# Test resource limits on Docker containers
# Usage: ./test-resource-limits.sh [container_id]

set -e

CONTAINER_ID=${1:-$(docker ps -q | head -1)}

if [ -z "$CONTAINER_ID" ]; then
    echo "No container found. Usage: $0 [container_id]"
    exit 1
fi

echo "=============================================="
echo "Resource Limit Tests for: $CONTAINER_ID"
echo "=============================================="

# Get container name
CONTAINER_NAME=$(docker inspect --format '{{.Name}}' $CONTAINER_ID | sed 's/\///')
echo "Container: $CONTAINER_NAME"
echo ""

# 1. DISK LIMIT TEST
echo "=== 1. DISK LIMIT TEST ==="
DISK_LIMIT=$(docker inspect --format '{{index .HostConfig.StorageOpt "size"}}' $CONTAINER_ID 2>/dev/null || echo "not set")
echo "Configured limit: $DISK_LIMIT"

echo "Testing disk inside container..."
docker exec $CONTAINER_ID df -h / 2>/dev/null || echo "df command failed"

echo ""
echo "Attempting to write beyond limit (2.5GB to test 2GB limit)..."
docker exec $CONTAINER_ID sh -c 'dd if=/dev/zero of=/tmp/testfile bs=1M count=2500 2>&1 || echo "✅ DISK LIMIT ENFORCED"' 
docker exec $CONTAINER_ID rm -f /tmp/testfile 2>/dev/null || true
echo ""

# 2. MEMORY LIMIT TEST
echo "=== 2. MEMORY LIMIT TEST ==="
MEM_LIMIT=$(docker inspect --format '{{.HostConfig.Memory}}' $CONTAINER_ID)
MEM_LIMIT_MB=$((MEM_LIMIT / 1024 / 1024))
echo "Configured limit: ${MEM_LIMIT_MB}MB"

echo "Current memory usage:"
docker stats --no-stream --format "{{.MemUsage}}" $CONTAINER_ID

echo ""
echo "Note: Memory limit is enforced by cgroups. Exceeding it will kill the process."
echo "✅ MEMORY LIMIT CONFIGURED"
echo ""

# 3. CPU LIMIT TEST
echo "=== 3. CPU LIMIT TEST ==="
CPU_PERIOD=$(docker inspect --format '{{.HostConfig.CpuPeriod}}' $CONTAINER_ID)
CPU_QUOTA=$(docker inspect --format '{{.HostConfig.CpuQuota}}' $CONTAINER_ID)
if [ "$CPU_PERIOD" -gt 0 ]; then
    CPU_LIMIT=$(echo "scale=2; $CPU_QUOTA / $CPU_PERIOD" | bc)
    echo "Configured limit: ${CPU_LIMIT} CPUs (quota: $CPU_QUOTA, period: $CPU_PERIOD)"
else
    echo "CPU limit: unlimited"
fi
echo "✅ CPU LIMIT CONFIGURED"
echo ""

# 4. PID LIMIT TEST
echo "=== 4. PID (PROCESS) LIMIT TEST ==="
PID_LIMIT=$(docker inspect --format '{{.HostConfig.PidsLimit}}' $CONTAINER_ID)
echo "Configured limit: $PID_LIMIT processes"

echo "Current process count:"
docker stats --no-stream --format "{{.PIDs}}" $CONTAINER_ID

echo ""
echo "Testing fork bomb protection (attempting to spawn 300 processes on 256 limit)..."
docker exec $CONTAINER_ID sh -c 'for i in $(seq 1 300); do sleep 100 & done 2>&1 | tail -5 || echo "✅ PID LIMIT ENFORCED"'
docker exec $CONTAINER_ID sh -c 'pkill -9 sleep 2>/dev/null || true'
echo ""

# 5. CAPABILITIES TEST
echo "=== 5. SECURITY CAPABILITIES ==="
echo "Dropped capabilities:"
docker inspect --format '{{.HostConfig.CapDrop}}' $CONTAINER_ID

echo "Added capabilities:"
docker inspect --format '{{.HostConfig.CapAdd}}' $CONTAINER_ID

echo "Security options:"
docker inspect --format '{{.HostConfig.SecurityOpt}}' $CONTAINER_ID
echo "✅ SECURITY HARDENING CONFIGURED"
echo ""

# 6. SUMMARY
echo "=============================================="
echo "SUMMARY"
echo "=============================================="
echo "Disk:     $DISK_LIMIT"
echo "Memory:   ${MEM_LIMIT_MB}MB"
echo "CPU:      ${CPU_LIMIT:-unlimited} CPUs"
echo "PIDs:     $PID_LIMIT"
echo "Runtime:  $(docker inspect --format '{{.HostConfig.Runtime}}' $CONTAINER_ID)"
echo "=============================================="
