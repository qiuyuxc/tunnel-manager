import { api } from './index'

/** 面板存储的来源：unset 表示还没配置，env 表示环境变量接管，file 表示来自安装引导。 */
export type SetupSource = 'unset' | 'env' | 'file'

export type SetupDriver = 'sqlite' | 'postgres'

/** 已解析的存储配置（发往浏览器的副本，凭据已被抹掉）。 */
export interface SetupStorage {
  driver: SetupDriver
  path?: string
  /** PostgreSQL 连接串；结构化字段缺失时用它展示（凭据已脱敏）。 */
  url?: string
  postgres?: {
    host: string
    port: number
    database: string
    user: string
    ssl_mode: string
  }
}

/** 安装表单：字段拍平，浏览器不需要自己拼连接串。 */
export interface SetupStoragePayload {
  driver: SetupDriver
  path?: string
  host?: string
  port?: number
  database?: string
  user?: string
  password?: string
  ssl_mode?: string
}

export interface SetupStatus {
  needs_setup: boolean
  source: SetupSource
  /** 环境变量已接管存储配置，引导只能设置管理员账户。 */
  env_locked: boolean
  storage: SetupStorage
  /** 已存在数据库密码，留空表示沿用。 */
  has_secret: boolean
  data_dir: string
  setup_file: string
}

export interface SetupResult {
  ok: boolean
  error?: string
  message?: string
  restarting?: boolean
}

let cached: SetupStatus | null | undefined

/**
 * setupStatus 读取一次安装状态：已安装的面板没有这个接口（404），会被记成
 * “没有引导”，后续路由守卫不再重复请求。
 */
export async function setupStatus(refresh = false): Promise<SetupStatus | null> {
  if (!refresh && cached !== undefined) return cached
  try {
    const { data } = await api.get<SetupStatus>('/setup/status')
    cached = data
  } catch {
    cached = null
  }
  return cached
}

export async function testSetupStorage(storage: SetupStoragePayload): Promise<SetupResult> {
  const { data } = await api.post<SetupResult>('/setup/test', storage)
  return data
}

export async function completeSetup(
  storage: SetupStoragePayload,
  username: string,
  password: string
): Promise<SetupResult> {
  const { data } = await api.post<SetupResult>('/setup/complete', {
    storage,
    admin: { username, password },
  })
  return data
}

/**
 * 等待重启后的面板接管：安装接口消失（404）才说明新进程真的起来了。
 * 不能用 /api/health 判断——安装模式下的旧进程也会答健康检查，会在服务
 * 重启前就误判成功，把操作者留在过期的“待安装”状态里。
 */
export async function waitForPanelRestart(timeoutMs = 90000): Promise<boolean> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    try {
      const res = await api.get('/setup/status', { timeout: 4000, validateStatus: () => true })
      if (res.status !== 200) {
        cached = null
        return true
      }
    } catch {
      // 服务正在重启：连接被拒或超时都算还没起来，继续等。
    }
    await new Promise((resolve) => setTimeout(resolve, 1500))
  }
  cached = null
  return false
}
