#!/bin/bash
# 测试服务注册API的脚本

HOST=${1:-"localhost"}
PORT=${2:-"8080"}
BASE_URL="http://$HOST:$PORT/api/v1"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}开始测试服务注册API (${BASE_URL})${NC}"

# 随机生成服务ID
SERVICE_NAME="test-service-$(date +%s)"
SERVICE_IP="192.168.1.100"
SERVICE_PORT=8888

echo -e "\n${YELLOW}1. 注册服务 $SERVICE_NAME${NC}"
# 注册服务
REGISTER_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" -d "{
  \"name\": \"$SERVICE_NAME\",
  \"namespace\": \"default\",
  \"ip\": \"$SERVICE_IP\",
  \"port\": $SERVICE_PORT,
  \"tags\": [\"test\", \"api\"],
  \"metadata\": {
    \"version\": \"1.0.0\",
    \"environment\": \"test\"
  },
  \"ttl\": \"60s\"
}" $BASE_URL/services)

echo "响应: $REGISTER_RESPONSE"

# 提取服务ID
SERVICE_ID=$(echo $REGISTER_RESPONSE | grep -o '"service_id":"[^"]*' | cut -d'"' -f4)

if [ -z "$SERVICE_ID" ]; then
  echo -e "${RED}注册服务失败，无法获取服务ID${NC}"
  exit 1
fi

echo -e "${GREEN}服务已注册，ID: $SERVICE_ID${NC}"

echo -e "\n${YELLOW}2. 发送心跳${NC}"
# 发送心跳
HEARTBEAT_RESPONSE=$(curl -s -X PUT $BASE_URL/services/$SERVICE_ID/heartbeat)
echo "响应: $HEARTBEAT_RESPONSE"

echo -e "\n${YELLOW}3. 休眠2秒${NC}"
sleep 2

echo -e "\n${YELLOW}4. 再次发送心跳${NC}"
# 再次发送心跳
HEARTBEAT_RESPONSE=$(curl -s -X PUT $BASE_URL/services/$SERVICE_ID/heartbeat)
echo "响应: $HEARTBEAT_RESPONSE"

echo -e "\n${YELLOW}5. 注销服务${NC}"
# 注销服务
DEREGISTER_RESPONSE=$(curl -s -X DELETE $BASE_URL/services/$SERVICE_ID)
echo "响应: $DEREGISTER_RESPONSE"

echo -e "\n${YELLOW}6. 尝试发送心跳到已注销的服务${NC}"
# 尝试发送心跳到已注销的服务
HEARTBEAT_RESPONSE=$(curl -s -X PUT $BASE_URL/services/$SERVICE_ID/heartbeat)
echo "响应: $HEARTBEAT_RESPONSE"

echo -e "\n${GREEN}测试完成${NC}" 