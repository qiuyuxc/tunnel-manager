#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=============================="
echo "  Tunnel Manager 安装/更新"
echo "=============================="

# Check dependencies
command -v docker >/dev/null 2>&1 || { echo "❌ 未安装 Docker，请先安装"; exit 1; }
if docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE="docker-compose"
else
  echo "❌ 未安装 Docker Compose"; exit 1
fi

command -v git >/dev/null 2>&1 || { echo "❌ 未安装 Git，请先安装"; exit 1; }

# If not inside the repo (no docker-compose.yml), clone it first
if [ ! -f docker-compose.yml ]; then
  echo ""
  echo "📥 首次使用，下载项目..."
  # Ask mirror first since clone needs it
  echo "🌐 选择下载源："
  echo "  1) 国内加速 (ghfast.top)"
  echo "  2) GitHub 直连"
  read -p "请选择 [1/2，默认1]: " CLONE_MIRROR
  CLONE_MIRROR=${CLONE_MIRROR:-1}
  if [ "$CLONE_MIRROR" = "1" ]; then
    CLONE_URL="https://ghfast.top/https://github.com/qiuyuxc/tunnel-manager.git"
  else
    CLONE_URL="https://github.com/qiuyuxc/tunnel-manager.git"
  fi
  git clone "$CLONE_URL"
  cd tunnel-manager
  export MIRROR_CHOICE="$CLONE_MIRROR"
  exec bash install.sh "$@"
fi

# Mirror source selection (skip if already chosen during clone)
if [ -z "$MIRROR_CHOICE" ]; then
  echo ""
  echo "🌐 选择镜像源："
  echo "  1) 国内镜像 (npmmirror / goproxy.cn / 阿里云 Alpine / GitHub 加速)"
  echo "  2) 官方源 (npmjs.org / proxy.golang.org / Alpine CDN / GitHub 直连)"
  read -p "请选择 [1/2，默认1]: " MIRROR_CHOICE
  MIRROR_CHOICE=${MIRROR_CHOICE:-1}
fi

if [ "$MIRROR_CHOICE" = "1" ]; then
  export NPM_REGISTRY=https://registry.npmmirror.com
  export GOPROXY=https://goproxy.cn,direct
  export ALPINE_MIRROR=https://mirrors.aliyun.com/alpine/
  GITHUB_PREFIX="https://ghfast.top/"
  echo "✅ 使用国内镜像源"
else
  GITHUB_PREFIX=""
  echo "✅ 使用官方源"
fi
REPO_URL="${GITHUB_PREFIX}https://github.com/qiuyuxc/tunnel-manager.git"

# Detect install or update
if [ -f .env ]; then
  echo ""
  echo "📦 检测到已有配置，执行更新..."
  if [ -d .git ]; then
    git remote set-url origin "$REPO_URL" 2>/dev/null || git remote add origin "$REPO_URL"
    echo "🔄 拉取最新代码..."
    git pull origin main
  fi
  if ! grep -q '^APP_ENCRYPTION_KEY=.' .env; then
    APP_ENCRYPTION_KEY=$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\n')
    if grep -q '^APP_ENCRYPTION_KEY=' .env; then
      sed -i "s|^APP_ENCRYPTION_KEY=.*|APP_ENCRYPTION_KEY=${APP_ENCRYPTION_KEY}|" .env
    else
      printf '\nAPP_ENCRYPTION_KEY=%s\n' "$APP_ENCRYPTION_KEY" >> .env
    fi
  fi
  for OAUTH_VAR in CF_OAUTH_CLIENT_ID CF_OAUTH_CLIENT_SECRET CF_OAUTH_REDIRECT_URI CF_OAUTH_SCOPES; do
    if ! grep -q "^${OAUTH_VAR}=" .env; then
      printf '%s=\n' "$OAUTH_VAR" >> .env
    fi
  done
  source .env
else
  echo ""
  echo "📝 首次运行，配置环境变量："
  echo "推荐使用 Cloudflare OAuth，API Token 仅作为旧版兼容方式。"
  read -p "CF_OAUTH_CLIENT_ID (推荐，留空则使用 API Token): " CF_OAUTH_CLIENT_ID
  CF_OAUTH_CLIENT_SECRET=""
  CF_OAUTH_REDIRECT_URI=""
  CF_TOKEN=""
  CF_AID=""
  if [ -n "$CF_OAUTH_CLIENT_ID" ]; then
    read -p "CF_OAUTH_CLIENT_SECRET: " CF_OAUTH_CLIENT_SECRET
    read -p "CF_OAUTH_REDIRECT_URI (例如 https://example.com/api/cloudflare/oauth/callback): " CF_OAUTH_REDIRECT_URI
  else
    read -p "CF_API_TOKEN: " CF_TOKEN
    read -p "CF_ACCOUNT_ID: " CF_AID
  fi
  read -p "API_KEY (可选，直接回车跳过): " API_KEY
  read -p "ADMIN_PASSWORD (可选，直接回车随机生成): " ADMIN_PASS
  APP_ENCRYPTION_KEY=$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\n')

  cat > .env <<EOF
CF_API_TOKEN=${CF_TOKEN}
CF_ACCOUNT_ID=${CF_AID}
CF_OAUTH_CLIENT_ID=${CF_OAUTH_CLIENT_ID}
CF_OAUTH_CLIENT_SECRET=${CF_OAUTH_CLIENT_SECRET}
CF_OAUTH_REDIRECT_URI=${CF_OAUTH_REDIRECT_URI}
CF_OAUTH_SCOPES=
API_KEY=${API_KEY}
ADMIN_PASSWORD=${ADMIN_PASS}
APP_ENCRYPTION_KEY=${APP_ENCRYPTION_KEY}
EOF
  echo "✅ .env 已生成"
fi

# Create data directory for persistence
mkdir -p data

# Build and start
echo ""
echo "🔨 构建镜像..."
DOCKER_BUILDKIT=1 $COMPOSE build --no-cache --progress=plain

echo ""
echo "🚀 启动服务..."
$COMPOSE up -d --force-recreate

echo ""
echo "=============================="
echo "  ✅ 部署完成！"
echo "  访问: http://$(hostname -I | awk '{print $1}'):8080"
if [ -n "$CF_OAUTH_CLIENT_ID" ]; then
  echo "  登录后请前往“全局设置”完成 Cloudflare OAuth 授权"
fi
echo ""

# Show password hint
if [ -z "$ADMIN_PASS" ]; then
  echo "  ⚠️  未设置密码，已自动生成"
  echo "  查看日志获取初始密码:"
  echo "    $COMPOSE logs | grep 密码"
else
  echo "  🔑 管理员密码: $ADMIN_PASS"
fi

echo ""
echo "  常用命令:"
echo "    查看日志: $COMPOSE logs -f"
echo "    停止服务: $COMPOSE down"
echo "    重启服务: $COMPOSE restart"
echo "    重置密码: $COMPOSE exec tunnel-manager ./tunnel-manager --reset-password"
echo "    设置密码: $COMPOSE exec tunnel-manager ./tunnel-manager --set-password=新密码"
echo "    更新服务: $0"
echo "=============================="
