#!/bin/bash

# 设置变量
DNS_PORT=${DNS_PORT:-6553}
DNS_SERVER="127.0.0.1"
DOMAIN="service.test"
API_PORT=${API_PORT:-8080}
SERVICE_NAME=${SERVICE_NAME:-"app"}
SERVICE_IP=${SERVICE_IP:-"192.168.1.100"}
SERVICE_PORT=${SERVICE_PORT:-8080}

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Kong网关DNS服务发现系统测试脚本${NC}"
echo "==================================="

# 检查DNS服务器是否运行
echo -e "${YELLOW}检查DNS服务器状态...${NC}"
nc -z -w2 $DNS_SERVER $DNS_PORT
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: DNS服务器未运行或端口 $DNS_PORT 不可访问${NC}"
    echo "请确保服务已启动: go run cmd/discovery/main.go -config configs/config.test.yaml"
    exit 1
fi
echo -e "${GREEN}DNS服务器运行正常 (端口 $DNS_PORT)${NC}"

# 注册测试服务
echo -e "\n${YELLOW}注册测试服务...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:$API_PORT/api/v1/services -H "Content-Type: application/json" -d "{
  \"name\": \"$SERVICE_NAME\",
  \"ip\": \"$SERVICE_IP\",
  \"port\": $SERVICE_PORT,
  \"tags\": [\"test\"],
  \"metadata\": {\"env\": \"test\"}
}")

if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到API服务器${NC}"
    exit 1
fi

SERVICE_ID=$(echo $REGISTER_RESPONSE | grep -o '"service_id":"[^"]*' | cut -d'"' -f4)
if [ -z "$SERVICE_ID" ]; then
    echo -e "${RED}错误: 服务注册失败${NC}"
    echo "API响应: $REGISTER_RESPONSE"
    exit 1
fi
echo -e "${GREEN}服务注册成功，ID: $SERVICE_ID${NC}"

# 等待DNS记录生效
echo -e "\n${YELLOW}等待DNS记录生效...${NC}"
sleep 2

# 测试A记录
echo -e "\n${YELLOW}测试A记录解析...${NC}"
A_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$DOMAIN A +short)
if [ -z "$A_RECORD" ]; then
    echo -e "${RED}错误: 无法解析A记录 $SERVICE_NAME.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$DOMAIN A
else
    echo -e "${GREEN}A记录解析成功: $SERVICE_NAME.$DOMAIN -> $A_RECORD${NC}"
    if [ "$A_RECORD" == "$SERVICE_IP" ]; then
        echo -e "${GREEN}A记录值正确匹配${NC}"
    else
        echo -e "${RED}A记录值不匹配 (期望: $SERVICE_IP, 实际: $A_RECORD)${NC}"
    fi
fi

# 测试SRV记录
echo -e "\n${YELLOW}测试SRV记录解析...${NC}"
SRV_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$DOMAIN SRV +short)
if [ -z "$SRV_RECORD" ]; then
    echo -e "${RED}错误: 无法解析SRV记录 _$SERVICE_NAME._tcp.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$DOMAIN SRV
else
    echo -e "${GREEN}SRV记录解析成功: _$SERVICE_NAME._tcp.$DOMAIN -> $SRV_RECORD${NC}"
    SRV_PORT=$(echo $SRV_RECORD | cut -d' ' -f3)
    if [ "$SRV_PORT" == "$SERVICE_PORT" ]; then
        echo -e "${GREEN}SRV记录端口正确匹配${NC}"
    else
        echo -e "${RED}SRV记录端口不匹配 (期望: $SERVICE_PORT, 实际: $SRV_PORT)${NC}"
    fi
fi

# 注销测试服务
echo -e "\n${YELLOW}注销测试服务...${NC}"
DEREGISTER_RESPONSE=$(curl -s -X DELETE http://localhost:$API_PORT/api/v1/services/$SERVICE_ID)
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到API服务器${NC}"
    exit 1
fi

echo -e "${GREEN}测试完成${NC}"
echo "===================================" 