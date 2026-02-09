#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

pass() { echo -e "${GREEN}✓ PASS${NC}: $1"; ((PASS++)); }
fail() { echo -e "${RED}✗ FAIL${NC}: $1 — $2"; ((FAIL++)); }

API="http://localhost:8080"
PG_CMD="docker compose exec -T postgres psql -U postgres -d fleet -t -A"
SEED_COUNT=5
SEED_INTERVAL=2
SEED_ROUNDS=3

# Generate 5 random vehicle IDs
generate_vehicle_id() {
    local letters="ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    local l1=${letters:$((RANDOM % 26)):1}
    local digits=$(printf "%04d" $((RANDOM % 10000)))
    local s1=${letters:$((RANDOM % 26)):1}
    local s2=${letters:$((RANDOM % 26)):1}
    local s3=${letters:$((RANDOM % 26)):1}
    echo "${l1}${digits}${s1}${s2}${s3}"
}

random_lat() { echo "-90 + $RANDOM / 32767 * 180" | bc -l | head -c 10; }
random_lon() { echo "-180 + $RANDOM / 32767 * 360" | bc -l | head -c 10; }

VEHICLE_IDS=()
for i in $(seq 1 $SEED_COUNT); do
    VEHICLE_IDS+=("$(generate_vehicle_id)")
done

echo -e "${YELLOW}=== Fleet Tracking Integration Tests ===${NC}"
echo -e "Seeded vehicle IDs: ${VEHICLE_IDS[*]}"
echo ""

# -------------------------------------------------------
# 1. Docker Compose — all services running
# -------------------------------------------------------
echo -e "${YELLOW}[Test 1] Docker Compose — all services healthy${NC}"
SERVICES=("postgres" "mosquitto" "rabbitmq" "server")
ALL_UP=true
for svc in "${SERVICES[@]}"; do
    STATUS=$(docker compose ps --format json "$svc" 2>/dev/null | jq -r '.State' 2>/dev/null || echo "missing")
    if [ "$STATUS" != "running" ]; then
        fail "Docker Compose" "$svc is $STATUS"
        ALL_UP=false
    fi
done
if [ "$ALL_UP" = true ]; then
    pass "Docker Compose — all services running"
fi

echo ""

# -------------------------------------------------------
# 2. Seed MQTT — publish location data for all vehicles
# -------------------------------------------------------
echo -e "${YELLOW}[Test 2] MQTT Publication — seeding ${SEED_COUNT} vehicles over ${SEED_ROUNDS} rounds (${SEED_INTERVAL}s interval)${NC}"

MQTT_OK=true
for round in $(seq 1 $SEED_ROUNDS); do
    # Pick a random vehicle from the pool each round
    RANDOM_IDX=$((RANDOM % SEED_COUNT))
    VID="${VEHICLE_IDS[$RANDOM_IDX]}"
    LAT=$(random_lat)
    LON=$(random_lon)
    TS=$(date +%s)

    MSG="{\"vehicle_id\":\"${VID}\",\"latitude\":${LAT},\"longitude\":${LON},\"timestamp\":${TS}}"
    TOPIC="/fleet/vehicle/${VID}/location"

    docker compose exec -T mosquitto mosquitto_pub \
        -t "$TOPIC" -m "$MSG" 2>/dev/null || { MQTT_OK=false; break; }

    echo "  [round $round] published to $TOPIC: $MSG"

    if [ "$round" -lt "$SEED_ROUNDS" ]; then
        sleep "$SEED_INTERVAL"
    fi
done

if [ "$MQTT_OK" = true ]; then
    pass "MQTT Publication — published $SEED_ROUNDS messages"
else
    fail "MQTT Publication" "failed to publish"
fi

# Verify MQTT subscriber received by listening for one more message
echo "  Verifying subscriber receives..."
VERIFY_IDX=$((RANDOM % SEED_COUNT))
VERIFY_VID="${VEHICLE_IDS[$VERIFY_IDX]}"
VERIFY_MSG="{\"vehicle_id\":\"${VERIFY_VID}\",\"latitude\":-6.2,\"longitude\":106.8,\"timestamp\":$(date +%s)}"
docker compose exec -T mosquitto mosquitto_pub \
    -t "/fleet/vehicle/${VERIFY_VID}/location" -m "$VERIFY_MSG" 2>/dev/null

echo ""

# Wait for data to flow through
sleep 3

# -------------------------------------------------------
# 3. PostgreSQL Storage — data is persisted
# -------------------------------------------------------
echo -e "${YELLOW}[Test 3] PostgreSQL Storage — data persisted${NC}"

ROW_COUNT=$(${PG_CMD} -c "SELECT COUNT(*) FROM vehicle_locations;" 2>/dev/null || echo "0")

if [ "$ROW_COUNT" -gt 0 ] 2>/dev/null; then
    pass "PostgreSQL Storage — $ROW_COUNT rows in vehicle_locations"
else
    fail "PostgreSQL Storage" "no rows found in vehicle_locations"
fi

DISTINCT_VEHICLES=$(${PG_CMD} -c "SELECT COUNT(DISTINCT vehicle_id) FROM vehicle_locations;" 2>/dev/null || echo "0")
if [ "$DISTINCT_VEHICLES" -gt 0 ] 2>/dev/null; then
    pass "PostgreSQL Storage — $DISTINCT_VEHICLES distinct vehicles stored"
else
    fail "PostgreSQL Storage" "no distinct vehicles found"
fi

echo ""

# -------------------------------------------------------
# 4. API — GET all vehicles
# -------------------------------------------------------
echo -e "${YELLOW}[Test 4] API — GET /vehicles${NC}"

HTTP_CODE=$(curl -s -o /tmp/api_vehicles.json -w '%{http_code}' "${API}/vehicles" 2>/dev/null || echo "000")

if [ "$HTTP_CODE" = "200" ]; then
    VCOUNT=$(jq 'length' /tmp/api_vehicles.json 2>/dev/null || echo "0")
    if [ "$VCOUNT" -gt 0 ] 2>/dev/null; then
        pass "API Get All Vehicles — returned $VCOUNT vehicles"
    else
        fail "API Get All Vehicles" "returned empty array"
    fi
else
    fail "API Get All Vehicles" "HTTP $HTTP_CODE"
fi

echo ""

# -------------------------------------------------------
# 5. API — GET latest location (pick a seeded vehicle)
# -------------------------------------------------------
echo -e "${YELLOW}[Test 5] API — GET /vehicles/{vehicle_id}/location${NC}"

TEST_VID="$VERIFY_VID"
HTTP_CODE=$(curl -s -o /tmp/api_latest.json -w '%{http_code}' "${API}/vehicles/${TEST_VID}/location" 2>/dev/null || echo "000")

if [ "$HTTP_CODE" = "200" ]; then
    VID=$(jq -r '.vehicle_id' /tmp/api_latest.json 2>/dev/null || echo "")
    LAT=$(jq -r '.latitude' /tmp/api_latest.json 2>/dev/null || echo "")
    TS=$(jq -r '.timestamp' /tmp/api_latest.json 2>/dev/null || echo "")

    if [ "$VID" = "$TEST_VID" ] && [ -n "$LAT" ] && [ "$TS" -gt 0 ] 2>/dev/null; then
        pass "API Latest Location — vehicle_id=$VID lat=$LAT timestamp=$TS"
    else
        fail "API Latest Location" "unexpected response: $(cat /tmp/api_latest.json)"
    fi
else
    fail "API Latest Location" "HTTP $HTTP_CODE — $(cat /tmp/api_latest.json 2>/dev/null)"
fi

echo ""

# -------------------------------------------------------
# 6. API — GET history with time range
# -------------------------------------------------------
echo -e "${YELLOW}[Test 6] API — GET /vehicles/{vehicle_id}/history${NC}"

NOW=$(date +%s)
START=$((NOW - 120))
END=$((NOW + 60))

HTTP_CODE=$(curl -s -o /tmp/api_history.json -w '%{http_code}' \
    "${API}/vehicles/${TEST_VID}/history?start=${START}&end=${END}" 2>/dev/null || echo "000")

if [ "$HTTP_CODE" = "200" ]; then
    COUNT=$(jq 'length' /tmp/api_history.json 2>/dev/null || echo "0")
    if [ "$COUNT" -gt 0 ] 2>/dev/null; then
        pass "API History — returned $COUNT records for vehicle $TEST_VID"
    else
        fail "API History" "returned empty array"
    fi
else
    fail "API History" "HTTP $HTTP_CODE — $(cat /tmp/api_history.json 2>/dev/null)"
fi

echo ""

# -------------------------------------------------------
# 7. RabbitMQ Geofence Event
# -------------------------------------------------------
echo -e "${YELLOW}[Test 7] RabbitMQ — Geofence alerts${NC}"

# Pick a random seeded vehicle and send it exactly to the geofence point
GEO_VID="${VEHICLE_IDS[$((RANDOM % SEED_COUNT))]}"
GEOFENCE_MSG="{\"vehicle_id\":\"${GEO_VID}\",\"latitude\":-6.2088,\"longitude\":106.8456,\"timestamp\":$(date +%s)}"
echo "  Publishing exact geofence location for $GEO_VID..."
docker compose exec -T mosquitto mosquitto_pub \
    -t "/fleet/vehicle/${GEO_VID}/location" -m "$GEOFENCE_MSG" 2>/dev/null

sleep 2

QUEUE_INFO=$(docker compose exec -T rabbitmq rabbitmqctl list_queues name messages 2>/dev/null || echo "")

if echo "$QUEUE_INFO" | grep -q "geofence_alerts"; then
    MSG_COUNT=$(echo "$QUEUE_INFO" | grep "geofence_alerts" | awk '{print $2}')
    if [ "$MSG_COUNT" -gt 0 ] 2>/dev/null; then
        pass "RabbitMQ Geofence — geofence_alerts queue has $MSG_COUNT messages"
    else
        fail "RabbitMQ Geofence" "no messages in geofence_alerts queue"
    fi
else
    fail "RabbitMQ Geofence" "geofence_alerts queue does not exist"
fi

echo ""

# -------------------------------------------------------
# Summary
# -------------------------------------------------------
echo -e "${YELLOW}==============================${NC}"
TOTAL=$((PASS + FAIL))
echo -e "Results: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC} out of ${TOTAL} checks"
echo -e "${YELLOW}==============================${NC}"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
