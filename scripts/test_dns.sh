#!/bin/bash

# 设置变量
DNS_PORT=${DNS_PORT:-6553}
DNS_SERVER="127.0.0.1"
DOMAIN="service.test"
API_PORT=${API_PORT:-8080}
ADMIN_PORT=${ADMIN_PORT:-9090}
SERVICE_NAME=${SERVICE_NAME:-"app"}
SERVICE_IP=${SERVICE_IP:-"192.168.1.100"}
SERVICE_PORT=${SERVICE_PORT:-8080}
NAMESPACE=${NAMESPACE:-"test-ns"}
DEFAULT_NS="default"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试计数
TOTAL_TESTS=0
PASSED_TESTS=0

# 测试函数
run_test() {
    local test_name="$1"
    local command="$2"
    local expected="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo -e "\n${YELLOW}测试: $test_name${NC}"
    local result=$(eval "$command")
    
    if [[ "$result" == *"$expected"* ]]; then
        echo -e "${GREEN}✓ 通过: $test_name${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ 失败: $test_name${NC}"
        echo -e "  期望包含: ${BLUE}$expected${NC}"
        echo -e "  实际结果: ${BLUE}$result${NC}"
    fi
}

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

# 检查管理API是否运行
echo -e "\n${YELLOW}检查管理API状态...${NC}"
curl -s http://localhost:$ADMIN_PORT/api/v1/health > /dev/null
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 管理API未运行或端口 $ADMIN_PORT 不可访问${NC}"
    exit 1
fi
echo -e "${GREEN}管理API运行正常 (端口 $ADMIN_PORT)${NC}"

# 检查注册API是否运行
echo -e "\n${YELLOW}检查注册API状态...${NC}"
curl -s http://localhost:$API_PORT/api/v1/health > /dev/null
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 注册API未运行或端口 $API_PORT 不可访问${NC}"
    exit 1
fi
echo -e "${GREEN}注册API运行正常 (端口 $API_PORT)${NC}"

# 创建测试命名空间
echo -e "\n${YELLOW}创建测试命名空间...${NC}"
NS_RESPONSE=$(curl -s -X POST http://localhost:$ADMIN_PORT/api/v1/namespaces -H "Content-Type: application/json" -d "{
  \"name\": \"$NAMESPACE\",
  \"description\": \"测试命名空间\"
}")

if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到管理API服务器${NC}"
    exit 1
fi

echo -e "${GREEN}命名空间创建成功: $NAMESPACE${NC}"

# 第1部分: 在自定义命名空间中注册服务
echo -e "\n${YELLOW}第1部分: 在自定义命名空间中注册服务${NC}"
echo -e "\n${YELLOW}注册测试服务 (命名空间: $NAMESPACE)...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:$API_PORT/api/v1/services -H "Content-Type: application/json" -d "{
  \"name\": \"$SERVICE_NAME\",
  \"namespace\": \"$NAMESPACE\",
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
echo -e "${GREEN}服务注册成功，ID: $SERVICE_ID，命名空间: $NAMESPACE${NC}"

# 等待DNS记录生效
echo -e "\n${YELLOW}等待DNS记录生效...${NC}"
sleep 2

# 测试带命名空间的A记录
echo -e "\n${YELLOW}测试带命名空间的A记录解析...${NC}"
NS_A_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$NAMESPACE.$DOMAIN A +short)
if [ -z "$NS_A_RECORD" ]; then
    echo -e "${RED}错误: 无法解析带命名空间的A记录 $SERVICE_NAME.$NAMESPACE.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$NAMESPACE.$DOMAIN A
else
    echo -e "${GREEN}带命名空间的A记录解析成功: $SERVICE_NAME.$NAMESPACE.$DOMAIN -> $NS_A_RECORD${NC}"
    if [ "$NS_A_RECORD" == "$SERVICE_IP" ]; then
        echo -e "${GREEN}A记录值正确匹配${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}A记录值不匹配 (期望: $SERVICE_IP, 实际: $NS_A_RECORD)${NC}"
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# 测试带命名空间的SRV记录
echo -e "\n${YELLOW}测试带命名空间的SRV记录解析...${NC}"
NS_SRV_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$NAMESPACE.$DOMAIN SRV +short)
if [ -z "$NS_SRV_RECORD" ]; then
    echo -e "${RED}错误: 无法解析带命名空间的SRV记录 _$SERVICE_NAME._tcp.$NAMESPACE.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$NAMESPACE.$DOMAIN SRV
else
    echo -e "${GREEN}带命名空间的SRV记录解析成功: _$SERVICE_NAME._tcp.$NAMESPACE.$DOMAIN -> $NS_SRV_RECORD${NC}"
    NS_SRV_PORT=$(echo $NS_SRV_RECORD | cut -d' ' -f3)
    if [ "$NS_SRV_PORT" == "$SERVICE_PORT" ]; then
        echo -e "${GREEN}SRV记录端口正确匹配${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}SRV记录端口不匹配 (期望: $SERVICE_PORT, 实际: $NS_SRV_PORT)${NC}"
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# 第2部分: 在默认命名空间中注册服务
echo -e "\n${YELLOW}第2部分: 在默认命名空间中注册服务${NC}"
echo -e "\n${YELLOW}注册测试服务 (命名空间: $DEFAULT_NS)...${NC}"
DEFAULT_SERVICE_IP="192.168.1.200"
DEFAULT_SERVICE_PORT=8888

REGISTER_DEFAULT_RESPONSE=$(curl -s -X POST http://localhost:$API_PORT/api/v1/services -H "Content-Type: application/json" -d "{
  \"name\": \"$SERVICE_NAME\",
  \"namespace\": \"$DEFAULT_NS\",
  \"ip\": \"$DEFAULT_SERVICE_IP\",
  \"port\": $DEFAULT_SERVICE_PORT,
  \"tags\": [\"default\"],
  \"metadata\": {\"env\": \"default\"}
}")

if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到API服务器${NC}"
    exit 1
fi

DEFAULT_SERVICE_ID=$(echo $REGISTER_DEFAULT_RESPONSE | grep -o '"service_id":"[^"]*' | cut -d'"' -f4)
if [ -z "$DEFAULT_SERVICE_ID" ]; then
    echo -e "${RED}错误: 默认命名空间服务注册失败${NC}"
    echo "API响应: $REGISTER_DEFAULT_RESPONSE"
    exit 1
fi
echo -e "${GREEN}服务注册成功，ID: $DEFAULT_SERVICE_ID，命名空间: $DEFAULT_NS${NC}"

# 等待DNS记录生效
echo -e "\n${YELLOW}等待DNS记录生效...${NC}"
sleep 2

# 测试标准A记录 (默认命名空间)
echo -e "\n${YELLOW}测试标准A记录解析 (默认命名空间)...${NC}"
A_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$DOMAIN A +short)
if [ -z "$A_RECORD" ]; then
    echo -e "${RED}错误: 无法解析A记录 $SERVICE_NAME.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$DOMAIN A
else
    echo -e "${GREEN}A记录解析成功: $SERVICE_NAME.$DOMAIN -> $A_RECORD${NC}"
    if [ "$A_RECORD" == "$DEFAULT_SERVICE_IP" ]; then
        echo -e "${GREEN}A记录值正确匹配${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}A记录值不匹配 (期望: $DEFAULT_SERVICE_IP, 实际: $A_RECORD)${NC}"
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# 测试显式指定默认命名空间的A记录
echo -e "\n${YELLOW}测试显式指定默认命名空间的A记录解析...${NC}"
DEFAULT_A_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$DEFAULT_NS.$DOMAIN A +short)
if [ -z "$DEFAULT_A_RECORD" ]; then
    echo -e "${RED}错误: 无法解析默认命名空间的A记录 $SERVICE_NAME.$DEFAULT_NS.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT $SERVICE_NAME.$DEFAULT_NS.$DOMAIN A
else
    echo -e "${GREEN}默认命名空间的A记录解析成功: $SERVICE_NAME.$DEFAULT_NS.$DOMAIN -> $DEFAULT_A_RECORD${NC}"
    if [ "$DEFAULT_A_RECORD" == "$DEFAULT_SERVICE_IP" ]; then
        echo -e "${GREEN}A记录值正确匹配${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}A记录值不匹配 (期望: $DEFAULT_SERVICE_IP, 实际: $DEFAULT_A_RECORD)${NC}"
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# 测试标准SRV记录 (默认命名空间)
echo -e "\n${YELLOW}测试标准SRV记录解析 (默认命名空间)...${NC}"
SRV_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$DOMAIN SRV +short)
if [ -z "$SRV_RECORD" ]; then
    echo -e "${RED}错误: 无法解析SRV记录 _$SERVICE_NAME._tcp.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$DOMAIN SRV
else
    echo -e "${GREEN}SRV记录解析成功: _$SERVICE_NAME._tcp.$DOMAIN -> $SRV_RECORD${NC}"
    SRV_PORT=$(echo $SRV_RECORD | cut -d' ' -f3)
    if [ "$SRV_PORT" == "$DEFAULT_SERVICE_PORT" ]; then
        echo -e "${GREEN}SRV记录端口正确匹配${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}SRV记录端口不匹配 (期望: $DEFAULT_SERVICE_PORT, 实际: $SRV_PORT)${NC}"
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# 测试带命名空间的SRV记录 (默认命名空间)
echo -e "\n${YELLOW}测试带命名空间的SRV记录解析 (默认命名空间)...${NC}"
DEFAULT_SRV_RECORD=$(dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$DEFAULT_NS.$DOMAIN SRV +short)
if [ -z "$DEFAULT_SRV_RECORD" ]; then
    echo -e "${RED}错误: 无法解析默认命名空间的SRV记录 _$SERVICE_NAME._tcp.$DEFAULT_NS.$DOMAIN${NC}"
    dig @$DNS_SERVER -p $DNS_PORT _$SERVICE_NAME._tcp.$DEFAULT_NS.$DOMAIN SRV
else
    echo -e "${GREEN}默认命名空间的SRV记录解析成功: _$SERVICE_NAME._tcp.$DEFAULT_NS.$DOMAIN -> $DEFAULT_SRV_RECORD${NC}"
    DEFAULT_SRV_PORT=$(echo $DEFAULT_SRV_RECORD | cut -d' ' -f3)
    if [ "$DEFAULT_SRV_PORT" == "$DEFAULT_SERVICE_PORT" ]; then
        echo -e "${GREEN}SRV记录端口正确匹配${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}SRV记录端口不匹配 (期望: $DEFAULT_SERVICE_PORT, 实际: $DEFAULT_SRV_PORT)${NC}"
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
fi

# 第3部分: 查询命名空间列表
echo -e "\n${YELLOW}第3部分: 查询命名空间列表${NC}"
echo -e "\n${YELLOW}获取命名空间列表...${NC}"
NS_LIST_RESPONSE=$(curl -s http://localhost:$ADMIN_PORT/api/v1/namespaces)
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到管理API服务器${NC}"
    exit 1
fi

echo "$NS_LIST_RESPONSE" | grep -q "\"name\":\"$DEFAULT_NS\""
if [ $? -eq 0 ]; then
    echo -e "${GREEN}默认命名空间存在: $DEFAULT_NS${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}未找到默认命名空间: $DEFAULT_NS${NC}"
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

echo "$NS_LIST_RESPONSE" | grep -q "\"name\":\"$NAMESPACE\""
if [ $? -eq 0 ]; then
    echo -e "${GREEN}测试命名空间存在: $NAMESPACE${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
else
    echo -e "${RED}未找到测试命名空间: $NAMESPACE${NC}"
fi
TOTAL_TESTS=$((TOTAL_TESTS + 1))

# 清理: 注销测试服务
echo -e "\n${YELLOW}清理: 注销测试服务...${NC}"
DEREGISTER_RESPONSE=$(curl -s -X DELETE http://localhost:$API_PORT/api/v1/services/$SERVICE_ID)
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到API服务器${NC}"
    exit 1
fi
echo -e "${GREEN}服务注销成功: $SERVICE_ID${NC}"

# 清理: 注销默认命名空间服务
echo -e "\n${YELLOW}清理: 注销默认命名空间服务...${NC}"
DEREGISTER_DEFAULT_RESPONSE=$(curl -s -X DELETE http://localhost:$API_PORT/api/v1/services/$DEFAULT_SERVICE_ID)
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到API服务器${NC}"
    exit 1
fi
echo -e "${GREEN}默认命名空间服务注销成功: $DEFAULT_SERVICE_ID${NC}"

# 清理: 删除测试命名空间
echo -e "\n${YELLOW}清理: 删除测试命名空间...${NC}"
NS_DELETE_RESPONSE=$(curl -s -X DELETE http://localhost:$ADMIN_PORT/api/v1/namespaces/$NAMESPACE)
if [ $? -ne 0 ]; then
    echo -e "${RED}错误: 无法连接到管理API服务器${NC}"
    exit 1
fi
echo -e "${GREEN}命名空间删除成功: $NAMESPACE${NC}"

# 测试报告
echo -e "\n${YELLOW}测试报告${NC}"
echo "==================================="
echo -e "总测试数: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "通过测试: ${GREEN}$PASSED_TESTS${NC}"
echo -e "失败测试: ${RED}$((TOTAL_TESTS - PASSED_TESTS))${NC}"

if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
    echo -e "\n${GREEN}所有测试都通过了!${NC}"
    exit 0
else
    echo -e "\n${RED}有测试失败，请检查上面的输出.${NC}"
    exit 1
fi 