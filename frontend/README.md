# Kong Discovery 前端界面

Kong Discovery服务发现系统的React前端界面。

## 功能特性

- 服务列表展示
- 服务详情查看
- DNS配置管理（待实现）
- 系统设置（待实现）

## 技术栈

- React 18+
- TypeScript
- React Router v7
- Ant Design组件库
- Axios HTTP客户端

## 开发环境

### 环境要求

- Node.js 18+
- npm 9+

### 启动开发环境

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

### 构建生产版本

```bash
npm run build
```

## 项目结构

```
frontend/
├── public/             # 静态资源
├── src/                # 源代码
│   ├── api/            # API客户端
│   ├── layouts/        # 布局组件
│   ├── pages/          # 页面组件
│   ├── router.tsx      # 路由配置
│   ├── main.tsx        # 入口文件
│   └── index.css       # 全局样式
├── package.json        # 项目配置
└── README.md           # 项目说明
```

## 接口说明

前端应用默认连接到`http://localhost:8080`的后端API。如需修改API地址，请更新`src/api/client.ts`文件中的`API_BASE_URL`常量。
