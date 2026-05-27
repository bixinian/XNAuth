<template>
  <section class="docs-page">
    <div class="docs-head">
      <div>
        <h2 class="page-title">对接文档</h2>
        <p class="page-subtitle">把客户端接入拆成密钥、加密信封、业务动作和调试案例，按步骤即可实现。</p>
      </div>
      <n-button type="primary" secondary @click="copyText('minimalPython')">
        <template #icon>
          <n-icon><CopyOutline /></n-icon>
        </template>
        复制最小案例
      </n-button>
    </div>

    <n-grid cols="1 m:4" responsive="screen" :x-gap="16" :y-gap="16">
      <n-gi v-for="item in essentials" :key="item.label">
        <div class="essential-item">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
          <small>{{ item.hint }}</small>
        </div>
      </n-gi>
    </n-grid>

    <n-alert type="info" :bordered="false" class="doc-alert">
      当前系统不是只给请求体签名，而是先用 X25519 派生 AES-GCM 密钥加密整个业务请求体；响应体也同样加密。Ed25519 用来证明请求来自持有设备私钥的客户端、响应来自持有服务端私钥的服务端。公钥可以公开，私钥不能公开。
    </n-alert>

    <n-tabs type="line" animated class="docs-tabs" default-value="flow">
      <n-tab-pane name="crypto" tab="加密原理">
        <n-card class="doc-card" :bordered="false">
          <div class="security-summary">
            <div>
              <span>先说结论</span>
              <h3>这套方案保护传输和设备连续身份，不把客户端变成不可伪造环境</h3>
              <p>
                应用级服务端私钥只保存在应用配置中，接口只返回对应应用公钥。客户端设备私钥保存在本机，它证明“还是这台已绑定设备”，但不能证明“这一定是官方客户端”。
              </p>
            </div>
            <ul>
              <li>拿到服务端公钥不能解密请求，也不能伪造服务端响应。</li>
              <li>拿到有效卡密的人，可以自己写客户端并首次绑定自己的设备公钥。</li>
              <li>设备完成绑定后，后续请求必须由已绑定私钥签名，否则会被拒绝。</li>
              <li>客户端私钥被复制后，攻击者可冒充该设备，所以生产客户端要用系统安全存储并减少明文落盘。</li>
            </ul>
          </div>

          <div class="doc-section-head">
            <div>
              <h3>密钥角色</h3>
              <p>先区分四类密钥，否则容易把“加密”和“签名”混在一起。</p>
            </div>
            <n-tag round :bordered="false" type="success">核心概念</n-tag>
          </div>

          <div class="role-grid">
            <div v-for="role in cryptoRoles" :key="role.title" class="role-item">
              <span>{{ role.owner }}</span>
              <strong>{{ role.title }}</strong>
              <p>{{ role.desc }}</p>
            </div>
          </div>

          <div class="doc-section-head compact">
            <div>
              <h3>一次请求发生了什么</h3>
              <p>客户端每次请求都重新生成临时 X25519 密钥，最终得到只在本次请求中使用的 AES 密钥。</p>
            </div>
          </div>
          <div class="crypto-flow">
            <div v-for="step in cryptoFlow" :key="step.title" class="crypto-step">
              <span>{{ step.no }}</span>
              <div>
                <strong>{{ step.title }}</strong>
                <p>{{ step.desc }}</p>
              </div>
            </div>
          </div>

          <div class="warning-grid">
            <n-alert v-for="item in boundaries" :key="item.title" type="warning" :bordered="false">
              <strong>{{ item.title }}</strong>
              <p>{{ item.desc }}</p>
            </n-alert>
          </div>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="flow" tab="落地流程">
        <n-card class="doc-card" :bordered="false">
          <n-alert type="warning" :bordered="false" class="doc-alert">
            落地时不要把 App Key、服务端公钥或客户端设备公钥当成秘密。真正的应用级服务端私钥不能通过任何接口、客户端包或公开仓库泄露；客户端设备私钥只能约束已绑定设备，不能阻止用户拿自己的有效卡密编写第三方客户端。
          </n-alert>

          <div class="doc-section-head">
            <div>
              <h3>总图：先配置，再绑定设备，最后长期心跳</h3>
              <p>把落地过程拆成后台配置、客户端初始化、首次授权、后续在线四段。每段只处理自己该保存的密钥和数据。</p>
            </div>
            <n-tag round :bordered="false" type="info">流程图解</n-tag>
          </div>

          <div class="flow-diagram">
            <div v-for="node in flowDiagram" :key="node.no" class="flow-node" :class="`lane-${node.lane}`">
              <div class="flow-node-top">
                <span>{{ node.no }}</span>
                <n-tag size="small" round :bordered="false" :type="node.tagType">{{ node.role }}</n-tag>
              </div>
              <strong>{{ node.title }}</strong>
              <p>{{ node.desc }}</p>
              <div class="flow-chips">
                <small v-for="chip in node.chips" :key="chip">{{ chip }}</small>
              </div>
            </div>
          </div>

          <div class="doc-section-head compact">
            <div>
              <h3>密钥去向：哪些能公开，哪些必须保护</h3>
              <p>客户端真正需要固定的是当前应用的服务端公钥；服务端私钥和客户端设备私钥都不能被第三方拿到。</p>
            </div>
          </div>

          <div class="key-map">
            <div v-for="item in keyMap" :key="item.name" class="key-map-row">
              <div>
                <span>{{ item.scope }}</span>
                <strong>{{ item.name }}</strong>
              </div>
              <dl>
                <div>
                  <dt>生成位置</dt>
                  <dd>{{ item.generated }}</dd>
                </div>
                <div>
                  <dt>保存位置</dt>
                  <dd>{{ item.stored }}</dd>
                </div>
                <div>
                  <dt>是否公开</dt>
                  <dd>{{ item.publicity }}</dd>
                </div>
                <div>
                  <dt>用途</dt>
                  <dd>{{ item.usage }}</dd>
                </div>
              </dl>
            </div>
          </div>

          <div class="doc-section-head compact">
            <div>
              <h3>一次 auth.verify 请求链路</h3>
              <p>首次授权会同时完成卡密校验、设备公钥绑定、会话创建和响应加密签名。</p>
            </div>
          </div>

          <div class="request-rail">
            <div v-for="item in requestRail" :key="item.no" class="request-step">
              <span>{{ item.no }}</span>
              <div>
                <strong>{{ item.title }}</strong>
                <p>{{ item.desc }}</p>
                <code>{{ item.payload }}</code>
              </div>
            </div>
          </div>

          <div class="flow-callout-grid">
            <div v-for="item in flowCallouts" :key="item.title" class="flow-callout">
              <strong>{{ item.title }}</strong>
              <p>{{ item.desc }}</p>
            </div>
          </div>

          <div class="doc-section-head compact">
            <div>
              <h3>文字步骤</h3>
              <p>按下面步骤实现客户端时，每一步都可以在后台日志或接口返回里验证。</p>
            </div>
          </div>

          <div class="step-list">
            <div v-for="step in flowSteps" :key="step.title" class="step-item">
              <span>{{ step.no }}</span>
              <div>
                <strong>{{ step.title }}</strong>
                <p>{{ step.desc }}</p>
              </div>
            </div>
          </div>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="secure" tab="信封格式">
        <n-card class="doc-card" :bordered="false">
          <div class="doc-section-head">
            <div>
              <h3>统一安全入口</h3>
              <p>所有正式客户端动作都通过同一个接口提交，业务动作由外层 `action` 分发。</p>
            </div>
            <n-tag round :bordered="false" type="success">推荐入口</n-tag>
          </div>

          <div class="endpoint-list">
            <div>
              <span>读取服务端公开配置</span>
              <code>GET /api/client/secure/config?app_key=your-app-key</code>
            </div>
            <div>
              <span>提交加密信封</span>
              <code>POST /api/client/secure</code>
            </div>
          </div>

          <div class="formula-list">
            <div v-for="formula in cryptoFormulas" :key="formula.title">
              <strong>{{ formula.title }}</strong>
              <pre><code>{{ formula.value }}</code></pre>
              <p>{{ formula.desc }}</p>
            </div>
          </div>
        </n-card>

        <div class="code-grid">
          <article v-for="block in secureBlocks" :key="block.key" class="code-panel">
            <header>
              <div>
                <span>{{ block.label }}</span>
                <strong>{{ block.title }}</strong>
              </div>
              <n-button size="small" secondary @click="copyText(block.key)">
                <template #icon>
                  <n-icon><CopyOutline /></n-icon>
                </template>
                复制
              </n-button>
            </header>
            <pre><code>{{ block.code }}</code></pre>
          </article>
        </div>
      </n-tab-pane>

      <n-tab-pane name="actions" tab="业务动作">
        <n-card class="doc-card" :bordered="false">
          <div class="action-list">
            <div v-for="action in actions" :key="action.name" class="action-item">
              <div class="action-title">
                <n-tag size="small" round :bordered="false" type="info">{{ action.name }}</n-tag>
                <strong>{{ action.title }}</strong>
              </div>
              <p>{{ action.desc }}</p>
              <dl>
                <div>
                  <dt>何时调用</dt>
                  <dd>{{ action.when }}</dd>
                </div>
                <div>
                  <dt>关键入参</dt>
                  <dd>{{ action.request }}</dd>
                </div>
                <div>
                  <dt>关键返回</dt>
                  <dd>{{ action.response }}</dd>
                </div>
                <div>
                  <dt>失败影响</dt>
                  <dd>{{ action.failure }}</dd>
                </div>
              </dl>
            </div>
          </div>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="cases" tab="参考案例">
        <div class="case-grid">
          <article v-for="block in caseBlocks" :key="block.key" class="code-panel">
            <header>
              <div>
                <span>{{ block.label }}</span>
                <strong>{{ block.title }}</strong>
              </div>
              <n-button size="small" secondary @click="copyText(block.key)">
                <template #icon>
                  <n-icon><CopyOutline /></n-icon>
                </template>
                复制
              </n-button>
            </header>
            <pre><code>{{ block.code }}</code></pre>
          </article>
        </div>
      </n-tab-pane>

      <n-tab-pane name="demo" tab="Python Demo">
        <n-card class="doc-card" :bordered="false">
          <div class="doc-section-head">
            <div>
              <h3>本地演示脚本</h3>
              <p>工作区已提供 `pythondemo/client_demo.py`，会打印本地业务 JSON、加密信封、服务端加密信封和解密后的响应 JSON。</p>
            </div>
            <n-button secondary @click="copyText('python')">
              <template #icon>
                <n-icon><CopyOutline /></n-icon>
              </template>
              复制命令
            </n-button>
          </div>
          <pre class="single-code"><code>{{ codeMap.python }}</code></pre>
          <n-alert type="warning" :bordered="false" class="doc-alert">
            生产客户端应内置或固定当前应用的 X25519 / Ed25519 公钥，不要依赖首次联网读取公钥；客户端设备 Ed25519 私钥只保存在本机，不能上传服务端。
          </n-alert>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="troubleshoot" tab="排错">
        <n-card class="doc-card" :bordered="false">
          <div class="trouble-list">
            <div v-for="item in troubleshooting" :key="item.error" class="trouble-item">
              <strong>{{ item.error }}</strong>
              <p>{{ item.reason }}</p>
              <small>{{ item.fix }}</small>
            </div>
          </div>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="notes" tab="上线注意">
        <n-card class="doc-card" :bordered="false">
          <div class="boundary-list">
            <div v-for="item in securityLimits" :key="item.title">
              <strong>{{ item.title }}</strong>
              <p>{{ item.desc }}</p>
            </div>
          </div>
          <ul class="note-list">
            <li v-for="note in notes" :key="note">{{ note }}</li>
          </ul>
        </n-card>
      </n-tab-pane>
    </n-tabs>
  </section>
</template>

<script setup lang="ts">
import { NAlert, NButton, NCard, NGi, NGrid, NIcon, NTabPane, NTabs, NTag, useMessage } from "naive-ui";
import { CopyOutline } from "@vicons/ionicons5";

const message = useMessage();

const essentials = [
  { label: "服务地址", value: "http(s)://host:18080", hint: "生产环境建议放到 HTTPS 和反向代理后" },
  { label: "正式入口", value: "/api/client/secure", hint: "所有业务动作都走加密信封" },
  { label: "加密算法", value: "X25519 + AES-GCM", hint: "请求体和响应体都加密" },
  { label: "签名算法", value: "Ed25519", hint: "客户端签请求，服务端签响应" }
];

const cryptoRoles = [
  {
    owner: "服务端",
    title: "X25519 固定密钥",
    desc: "每个应用在应用管理中单独配置 X25519 私钥，客户端只需要该应用公钥。/secure/config 只按 app_key 返回公钥，私钥不能下发。"
  },
  {
    owner: "服务端",
    title: "Ed25519 响应签名密钥",
    desc: "每个应用在应用管理中单独配置 Ed25519 私钥，客户端保存对应公钥，用来校验响应确实来自该应用密钥配置。"
  },
  {
    owner: "客户端",
    title: "Ed25519 设备密钥",
    desc: "客户端首次运行本地生成，私钥永不上传；它是设备身份凭据，不是防破解根密钥。授权时提交公钥，后续请求由服务端用已绑定公钥验签。"
  },
  {
    owner: "客户端",
    title: "X25519 临时密钥",
    desc: "每次请求新生成一次，只用于本次 ECDH。临时公钥放进信封，临时私钥请求结束后丢弃。"
  }
];

const cryptoFlow = [
  { no: "1", title: "客户端准备业务 JSON", desc: "例如 auth.verify 明文中包含 app_key、license_key、machine_code、device_public_key、client_version_code 和 nonce。" },
  { no: "2", title: "生成本次临时 X25519 密钥", desc: "客户端用临时私钥和服务端 X25519 公钥计算 shared_key，并把临时公钥放进外层信封。" },
  { no: "3", title: "HKDF 派生 AES 密钥", desc: "服务端和客户端使用相同 salt 和 info 从 shared_key 派生 32 字节 AES-GCM 密钥。" },
  { no: "4", title: "AES-GCM 加密业务 JSON", desc: "body_nonce 必须是 12 字节随机数；AAD 是外层信封元数据的 canonical string。" },
  { no: "5", title: "Ed25519 签名外层信封", desc: "签名内容不是明文，而是外层元数据加 ciphertext_sha256，防止密文或路由字段被替换。" },
  { no: "6", title: "服务端解密、验签、执行业务", desc: "服务端先解密得到明文，再按 action 分发；auth.verify 会保存设备公钥，后续请求使用该公钥验签。" },
  { no: "7", title: "服务端加密并签名响应", desc: "响应 JSON 也进入 AES-GCM 密文，服务端再用 Ed25519 私钥签名响应信封。" },
  { no: "8", title: "客户端验签后解密响应", desc: "客户端先校验服务端 Ed25519 签名，再用本次 AES 密钥解密响应体并处理 code/data。" }
];

const boundaries = [
  {
    title: "业务 JSON 会被加密",
    desc: "app_key、卡密、机器码等业务字段在 ciphertext 中，不以明文出现在 HTTP body 的业务部分。"
  },
  {
    title: "外层信封不加密",
    desc: "v、kid、action、timestamp、nonce、临时公钥等路由字段是明文，但会被 AAD 和 Ed25519 签名绑定，篡改会失败。"
  },
  {
    title: "不负责证明客户端不可修改",
    desc: "只要攻击者控制客户端环境，就能分析协议并写自己的客户端。服务端能做的是校验卡密、设备绑定、会话、版本、nonce 和风控数据。"
  }
];

const flowDiagram = [
  {
    no: "01",
    lane: "server",
    role: "后台",
    tagType: "info" as const,
    title: "创建应用",
    desc: "后台应用管理创建 App，系统为该应用生成独立的 kid、X25519 密钥对和 Ed25519 密钥对。",
    chips: ["App Key", "kid", "应用级服务端私钥"]
  },
  {
    no: "02",
    lane: "server",
    role: "后台",
    tagType: "info" as const,
    title: "创建卡密",
    desc: "在卡密管理里创建授权卡密，配置设备上限、到期时间、状态和备注。",
    chips: ["license_key", "设备上限", "过期时间"]
  },
  {
    no: "03",
    lane: "client",
    role: "客户端",
    tagType: "success" as const,
    title: "固定应用公钥",
    desc: "客户端内置或安全配置当前应用的 kid、server_x25519_public_key、server_ed25519_public_key。",
    chips: ["服务端 X25519 公钥", "服务端 Ed25519 公钥"]
  },
  {
    no: "04",
    lane: "client",
    role: "客户端",
    tagType: "success" as const,
    title: "生成设备密钥",
    desc: "客户端首次启动生成 Ed25519 设备密钥对，私钥保存本地，公钥放进首次授权明文 JSON。",
    chips: ["设备私钥本地保存", "设备公钥可上传"]
  },
  {
    no: "05",
    lane: "network",
    role: "请求",
    tagType: "warning" as const,
    title: "加密并签名 auth.verify",
    desc: "业务 JSON 先用 AES-GCM 加密，外层信封再用设备私钥签名，提交到 /api/client/secure。",
    chips: ["ciphertext", "device sign", "nonce"]
  },
  {
    no: "06",
    lane: "server",
    role: "服务端",
    tagType: "info" as const,
    title: "解密验签并绑定",
    desc: "服务端按 app_key 读取应用私钥解密，验证请求签名，校验卡密并绑定设备公钥。",
    chips: ["应用私钥解密", "设备公钥验签", "创建 session"]
  },
  {
    no: "07",
    lane: "network",
    role: "响应",
    tagType: "warning" as const,
    title: "加密并签名响应",
    desc: "授权结果、session_token、heartbeat_interval 进入响应密文，服务端用应用 Ed25519 私钥签名。",
    chips: ["response ciphertext", "server sign"]
  },
  {
    no: "08",
    lane: "client",
    role: "客户端",
    tagType: "success" as const,
    title: "验签解密并长运行",
    desc: "客户端用应用 Ed25519 公钥验签，再解密响应，之后按心跳间隔持续发送 heartbeat 和上报。",
    chips: ["验服务端签名", "保存 session_token", "30 秒心跳"]
  }
];

const keyMap = [
  {
    scope: "服务端应用密钥",
    name: "secure_x25519_private_key",
    generated: "后台创建应用或点击生成密钥",
    stored: "服务端应用配置",
    publicity: "绝不能公开",
    usage: "服务端解密客户端用应用 X25519 公钥加密的请求"
  },
  {
    scope: "服务端应用密钥",
    name: "secure_x25519_public_key",
    generated: "由应用 X25519 私钥推导",
    stored: "客户端内置或通过 secure/config 读取后固定",
    publicity: "可以公开",
    usage: "客户端参与 ECDH，派生本次请求的 AES-GCM 密钥"
  },
  {
    scope: "服务端应用密钥",
    name: "secure_ed25519_private_key",
    generated: "后台创建应用或点击生成密钥",
    stored: "服务端应用配置",
    publicity: "绝不能公开",
    usage: "服务端给响应信封签名"
  },
  {
    scope: "服务端应用密钥",
    name: "secure_ed25519_public_key",
    generated: "由应用 Ed25519 私钥推导",
    stored: "客户端内置或通过 secure/config 读取后固定",
    publicity: "可以公开",
    usage: "客户端验证响应确实来自当前应用服务端配置"
  },
  {
    scope: "客户端设备密钥",
    name: "device_ed25519_private_key",
    generated: "客户端首次启动本地生成",
    stored: "客户端本地安全存储",
    publicity: "绝不能公开",
    usage: "客户端给请求信封签名，证明是已绑定设备"
  },
  {
    scope: "客户端设备密钥",
    name: "device_public_key",
    generated: "由客户端设备私钥推导",
    stored: "首次 auth.verify 上传后由服务端保存",
    publicity: "可以公开",
    usage: "服务端验证后续请求签名，防止同机器码换密钥冒充"
  }
];

const requestRail = [
  {
    no: "C1",
    title: "客户端准备授权明文",
    desc: "明文只在客户端内存里存在，包含卡密、机器码、设备公钥和版本信息。",
    payload: "{ app_key, license_key, machine_code, device_public_key, client_version_code, nonce }"
  },
  {
    no: "C2",
    title: "客户端派生 AES 密钥",
    desc: "每次请求生成临时 X25519 私钥，用它和应用 X25519 公钥做 ECDH，再通过 HKDF 得到本次 AES 密钥。",
    payload: "AES key = HKDF(X25519(temp_private, server_x25519_public_key))"
  },
  {
    no: "C3",
    title: "客户端加密业务 JSON",
    desc: "auth.verify 明文 JSON 被 AES-GCM 加密成 ciphertext，外层只留下 action、app_key、nonce、临时公钥等路由字段。",
    payload: "ciphertext = AES-GCM(plaintext, request_aad)"
  },
  {
    no: "C4",
    title: "客户端签名外层信封",
    desc: "用设备 Ed25519 私钥签名外层元数据和 ciphertext_sha256，避免密文或 action 被替换。",
    payload: "sign = Ed25519(device_private_key, canonical(envelope + ciphertext_sha256))"
  },
  {
    no: "S1",
    title: "服务端选择应用密钥",
    desc: "服务端先读取外层 app_key，找到该应用的 kid、X25519 私钥、Ed25519 私钥。",
    payload: "app_key -> apps.secure_*"
  },
  {
    no: "S2",
    title: "服务端解密并执行业务",
    desc: "服务端用应用 X25519 私钥派生同一个 AES 密钥，解密得到明文，再校验卡密状态、设备上限和版本。",
    payload: "plaintext -> check license/device/version"
  },
  {
    no: "S3",
    title: "首次绑定设备公钥",
    desc: "auth.verify 成功时保存 device_public_key；后续 heartbeat、公告、更新、上报都必须用同一设备私钥签名。",
    payload: "machine_code_hash + device_public_key"
  },
  {
    no: "S4",
    title: "服务端返回加密响应",
    desc: "session_token 和 heartbeat_interval 也会进入响应密文，然后服务端用应用 Ed25519 私钥签名响应信封。",
    payload: "{ code, data } -> response ciphertext + server sign"
  },
  {
    no: "C5",
    title: "客户端验签后解密",
    desc: "客户端先用应用 Ed25519 公钥验响应签名，验签通过后再用本次 AES 密钥解密响应体。",
    payload: "verify(server_ed25519_public_key) -> decrypt(response)"
  },
  {
    no: "C6",
    title: "进入长期在线流程",
    desc: "客户端保存 session_token，按 heartbeat_interval 发送 auth.heartbeat，并可按需调用公告、更新、数据上报。",
    payload: "auth.heartbeat / announcements / update.check / collect.report"
  }
];

const flowCallouts = [
  {
    title: "为什么重生成密钥会影响客户端",
    desc: "客户端固定的是应用 kid 和两个服务端公钥。重新生成后，旧 kid 不匹配，旧 X25519 公钥加密的请求服务端无法解密，旧 Ed25519 公钥也无法验证新响应。"
  },
  {
    title: "为什么 secure/config 只给公钥",
    desc: "公钥用于加密和验签，本身不是秘密；真正必须保护的是服务端应用私钥和客户端设备私钥。"
  },
  {
    title: "为什么有效卡密仍可能被第三方客户端使用",
    desc: "首次绑定时设备公钥由客户端提交。拥有有效卡密的人可以用自己的设备密钥完成首次绑定，这属于授权系统边界，需要靠卡密状态、设备上限和风控继续约束。"
  }
];

const flowSteps = [
  { no: "01", title: "后台创建应用和卡密", desc: "在应用管理获取 App Key，在卡密管理创建卡密并设置设备上限、到期时间和状态。" },
  { no: "02", title: "客户端本地生成设备密钥", desc: "生成 Ed25519 设备密钥对，私钥写入本地安全存储，公钥用于 auth.verify 首次绑定。Python Demo 明文文件仅用于演示。" },
  { no: "03", title: "固定应用公钥", desc: "开发阶段可按 app_key 读取 /secure/config；生产客户端应内置该应用的 kid、server_x25519_public_key、server_ed25519_public_key，并校验与预期一致。" },
  { no: "04", title: "调用 auth.verify", desc: "加密授权请求，成功后保存 session_token 和 heartbeat_interval。失败时查看验证日志的失败原因。" },
  { no: "05", title: "循环发送 auth.heartbeat", desc: "按 heartbeat_interval 长运行心跳；心跳断开超过会话超时时间后后台会认为离线。" },
  { no: "06", title: "按需扩展动作", desc: "登录后可调用 announcements、update.check 和 collect.report，全部复用同一套安全信封。" }
];

const cryptoFormulas = [
  {
    title: "AES 密钥派生",
    value: `shared_key = X25519(client_ephemeral_private, server_x25519_public)
salt = SHA256("xnauth-secure-transport-salt:" + kid + client_ephemeral_public_raw + server_x25519_public_raw)
aes_key = HKDF-SHA256(shared_key, salt=salt, info="xnauth secure transport v1", length=32)`,
    desc: "client_ephemeral_public_raw 和 server_x25519_public_raw 都是 32 字节原始公钥，不是 Base64 字符串。"
  },
  {
    title: "请求 AAD",
    value: `request_aad = canonical({
  v, kid, action, app_key, device_id, timestamp, nonce, body_nonce, client_ephemeral_public_key
})`,
    desc: "AES-GCM 加密和解密时必须使用完全一致的 AAD。"
  },
  {
    title: "请求签名",
    value: `request_sign = Ed25519Sign(device_private_key, canonical({
  v, kid, action, app_key, device_id, timestamp, nonce, body_nonce,
  client_ephemeral_public_key, ciphertext_sha256
}))`,
    desc: "ciphertext_sha256 是密文原始字节的 SHA-256 十六进制。"
  },
  {
    title: "响应签名",
    value: `response_sign = Ed25519Sign(server_private_key, canonical({
  v, kid, request_nonce, nonce, timestamp, body_nonce, ciphertext_sha256
}))`,
    desc: "客户端必须先验签，再用本次请求派生的 aes_key 解密响应 ciphertext。"
  }
];

const actions = [
  {
    name: "auth.verify",
    title: "授权验证",
    desc: "首次授权、绑定或刷新设备，成功后返回卡密信息和 session_token。",
    when: "客户端启动、登录、需要重新授权时。",
    request: "app_key, license_key, machine_code, device_public_key, client_version, client_version_code, nonce",
    response: "license, session.session_token, session.heartbeat_interval, server_time, client_nonce",
    failure: "无法继续使用软件；后台验证日志会记录失败原因。"
  },
  {
    name: "auth.heartbeat",
    title: "在线心跳",
    desc: "客户端长运行期间保持在线状态，服务端会刷新会话和设备最后出现时间。",
    when: "auth.verify 成功后按 heartbeat_interval 循环发送。",
    request: "app_key, license_key, session_token, machine_code, client_version, client_version_code, nonce",
    response: "server_time, client_nonce",
    failure: "客户端应停止或重新 auth.verify；后台会话可能变为离线或超时。"
  },
  {
    name: "announcements",
    title: "公告获取",
    desc: "获取当前应用有效公告，客户端可按 popup 字段决定是否弹窗。",
    when: "启动后、进入主界面前、用户手动刷新公告时。",
    request: "app_key, license_key, machine_code, nonce",
    response: "announcements, payload_digest, server_time, client_nonce",
    failure: "不影响授权，但客户端不能显示最新公告。"
  },
  {
    name: "update.check",
    title: "更新检查",
    desc: "读取最新启用版本，并结合应用管理中的最低登录版本和强制更新策略返回更新状态。",
    when: "启动后、授权成功后或用户手动检查更新时。",
    request: "app_key, license_key, machine_code, client_version, client_version_code, nonce",
    response: "update.has_update, latest_version_code, force_update, download_url, changelog",
    failure: "不影响当前授权，但客户端不能感知新版本。"
  },
  {
    name: "collect.report",
    title: "数据上报",
    desc: "上报后台已配置的收集字段，未配置字段会被忽略。",
    when: "授权成功后按业务周期上报，例如 60 秒一次或关键事件发生时。",
    request: "app_key, license_key, machine_code, event, data, nonce",
    response: "record_id, saved_fields",
    failure: "不影响授权，但数据统计和收集记录不会更新。"
  }
];

const codeMap: Record<string, string> = {
  config: `GET /api/client/secure/config?app_key=your-app-key

该接口按 App Key 返回当前应用公开配置，不返回服务端私钥。

{
  "code": 0,
  "message": "ok",
  "data": {
    "enabled": true,
    "kid": "app-secure-key-id",
    "algorithm": "X25519-HKDF-SHA256/AES-256-GCM + Ed25519",
    "app_key": "your-app-key",
    "server_x25519_public_key": "base64-32字节服务端X25519公钥",
    "server_ed25519_public_key": "base64-32字节服务端Ed25519公钥",
    "timestamp_skew_seconds": 120
  }
}`,
  canonical: `canonical(data):
1. 按 key 字典序升序排序。
2. 每项转成 key=value。
3. key 和 value 都做 query escape。
4. 多项使用 & 连接。

示例:
{
  "action": "auth.verify",
  "nonce": "n1",
  "v": 1
}

输出:
action=auth.verify&nonce=n1&v=1`,
  envelope: `POST /api/client/secure

{
  "v": 1,
  "kid": "app-secure-key-id",
  "action": "auth.verify",
  "app_key": "your-app-key",
  "device_id": 0,
  "timestamp": 1770000000,
  "nonce": "verify-uuid",
  "body_nonce": "base64-12字节随机数",
  "client_ephemeral_public_key": "base64-32字节临时X25519公钥",
  "ciphertext": "base64-AES-GCM密文和tag",
  "sign": "base64-Ed25519签名"
}`,
  verify: `{
  "app_key": "your-app-key",
  "license_key": "20位卡密",
  "machine_code": "stable-machine-id",
  "device_name": "Windows Client",
  "device_public_key": "base64-客户端Ed25519公钥",
  "client_version": "1.0.0",
  "client_version_code": 100,
  "nonce": "verify-uuid"
}`,
  heartbeat: `{
  "app_key": "your-app-key",
  "license_key": "20位卡密",
  "session_token": "session_xxx",
  "machine_code": "stable-machine-id",
  "client_version": "1.0.0",
  "client_version_code": 100,
  "nonce": "heartbeat-uuid"
}`,
  response: `{
  "v": 1,
  "kid": "app-secure-key-id",
  "request_nonce": "verify-uuid",
  "nonce": "server-random",
  "timestamp": 1770000001,
  "body_nonce": "base64-12字节随机数",
  "ciphertext": "base64-加密后的{code,message,data}",
  "sign": "base64-服务端Ed25519签名"
}`,
  minimalPython: `# 参考 pythondemo/client_demo.py 的最小请求顺序
crypto = load_crypto_modules()
secure_config = load_secure_config()
device_key = load_or_create_device_key(crypto, DEVICE_KEY_FILE)
client = SecureClient(crypto, secure_config, device_key)

verify_data = client.request("auth.verify", app_key, {
    "app_key": app_key,
    "license_key": license_key,
    "machine_code": machine_code,
    "device_name": "Python Secure Demo",
    "device_public_key": device_key.public_key_base64,
    "client_version": "1.0.0",
    "client_version_code": 100,
    "nonce": new_nonce("verify"),
})
session_token = verify_data["session"]["session_token"]`,
  pythonEncrypt: `ephemeral_private = x25519.X25519PrivateKey.generate()
ephemeral_public = ephemeral_private.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)
server_public = server_x25519_public_key.public_bytes(Encoding.Raw, PublicFormat.Raw)
shared_key = ephemeral_private.exchange(server_x25519_public_key)

salt = sha256(b"xnauth-secure-transport-salt:" + kid.encode() + ephemeral_public + server_public).digest()
aes_key = HKDF(SHA256(), 32, salt, b"xnauth secure transport v1").derive(shared_key)

body_nonce = os.urandom(12)
ciphertext = AESGCM(aes_key).encrypt(body_nonce, plaintext_json, request_aad.encode())`,
  pythonSign: `sign_data = dict(envelope)
sign_data.pop("sign", None)
sign_data.pop("ciphertext", None)
sign_data["ciphertext_sha256"] = sha256(ciphertext).hexdigest()
envelope["sign"] = b64encode(device_private_key.sign(canonical(sign_data).encode()))`,
  pythonResponse: `ciphertext = base64.b64decode(response["ciphertext"])
sign_payload = {
    "v": response["v"],
    "kid": response["kid"],
    "request_nonce": response["request_nonce"],
    "nonce": response["nonce"],
    "timestamp": response["timestamp"],
    "body_nonce": response["body_nonce"],
    "ciphertext_sha256": sha256(ciphertext).hexdigest(),
}
server_ed25519_public_key.verify(base64.b64decode(response["sign"]), canonical(sign_payload).encode())
plaintext = AESGCM(aes_key).decrypt(base64.b64decode(response["body_nonce"]), ciphertext, canonical(response_aad).encode())`,
  python: `cd C:\\Users\\xinian\\Documents\\网络验证系统
pip install -r .\\pythondemo\\requirements.txt
$env:XNAUTH_BASE_URL="http://127.0.0.1:18080"
$env:XNAUTH_APP_KEY="your-app-key"
python .\\pythondemo\\client_demo.py`
};

const secureBlocks = [
  { key: "config", label: "配置", title: "读取服务端安全配置", code: codeMap.config },
  { key: "canonical", label: "规范化", title: "canonical string 规则", code: codeMap.canonical },
  { key: "envelope", label: "信封", title: "统一加密请求格式", code: codeMap.envelope },
  { key: "verify", label: "业务", title: "auth.verify 明文 JSON", code: codeMap.verify },
  { key: "heartbeat", label: "业务", title: "auth.heartbeat 明文 JSON", code: codeMap.heartbeat },
  { key: "response", label: "响应", title: "服务端加密响应信封", code: codeMap.response }
];

const caseBlocks = [
  { key: "minimalPython", label: "最小流程", title: "授权并拿 session_token", code: codeMap.minimalPython },
  { key: "pythonEncrypt", label: "加密", title: "Python 派生 AES 密钥并加密", code: codeMap.pythonEncrypt },
  { key: "pythonSign", label: "签名", title: "Python 生成请求签名", code: codeMap.pythonSign },
  { key: "pythonResponse", label: "响应", title: "Python 验签并解密响应", code: codeMap.pythonResponse }
];

const troubleshooting = [
  {
    error: "invalid_secure_request",
    reason: "外层信封无法解密或基础字段不合法，常见于 kid、公钥、body_nonce、ciphertext 或 AAD 不一致。",
    fix: "确认 server_x25519_public_key 使用原始 32 字节 Base64；request_aad 字段和排序必须与文档一致。"
  },
  {
    error: "request_signature_invalid",
    reason: "AES 解密成功，但 Ed25519 请求签名校验失败。",
    fix: "确认签名内容使用 ciphertext_sha256，不要直接签 ciphertext；确认设备私钥和提交的 device_public_key 是同一对。"
  },
  {
    error: "nonce_mismatch",
    reason: "业务 JSON 内的 nonce 和外层信封 nonce 不一致。",
    fix: "生成一次 nonce，同时写入明文业务 JSON 和外层 envelope。"
  },
  {
    error: "replay_request",
    reason: "相同设备在重放窗口内重复使用了同一个 nonce。",
    fix: "每次请求使用 uuid 或足够随机的 nonce，不要缓存请求体重复发送。"
  },
  {
    error: "device_key_mismatch",
    reason: "同一机器码已经绑定过设备公钥，但这次提交了不同的设备公钥。",
    fix: "不要删除或重置本地设备私钥；如确需换密钥，后台先解绑或重置设备绑定。"
  },
  {
    error: "version_not_supported",
    reason: "client_version_code 小于应用管理中的最低登录版本，或版本号非法。",
    fix: "更新客户端版本号，或在应用管理中调整最低登录版本。"
  }
];

const notes = [
  "服务端 X25519 私钥和 Ed25519 私钥已经改为应用级配置；每个应用应使用不同密钥，不能进入前端、客户端或公开仓库。",
  "客户端设备私钥不是不可提取的秘密。桌面客户端建议使用 Windows DPAPI、macOS Keychain、Linux Secret Service 等系统安全存储。",
  "生产客户端应固定当前应用公钥；如果完全依赖首次访问 /secure/config，首次连接被劫持时可能被替换成攻击者公钥。",
  "machine_code 必须稳定，同一设备重启后应保持一致；服务端保存的是机器码哈希。",
  "nonce 必须每次请求唯一，服务端会记录成功请求的 nonce 防止重放。",
  "client_version_code 必须是递增整数；低于应用管理中的最低登录版本会被拒绝授权。",
  "卡密的设备上限控制可绑定设备数量，在线状态由心跳和会话超时时间共同决定。",
  "collect.report 只保存后台收集字段中已启用的字段，字段值过长会被截断。",
  "兼容旧接口仍存在，但正式客户端应优先接入 /api/client/secure。"
];

const securityLimits = [
  {
    title: "能防住什么",
    desc: "防止业务请求体和响应体被旁路读取；防止 action、nonce、timestamp、密文被篡改；防止同一设备请求被重放；防止已绑定设备被换公钥冒充。"
  },
  {
    title: "防不住什么",
    desc: "防不住拥有有效卡密的人自行实现客户端；防不住客户端环境被完全控制后读取本地设备私钥；防不住应用级服务端私钥泄露。"
  },
  {
    title: "服务端应该继续做什么",
    desc: "把授权判断都放在服务端，包括卡密状态、设备上限、会话心跳、版本策略、数据上报异常、封禁和解绑。客户端只负责展示和执行。"
  }
];

async function copyText(key: string) {
  const text = codeMap[key];
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    message.success("已复制");
  } catch {
    message.warning("浏览器限制自动复制，请手动选择代码");
  }
}
</script>

<style scoped>
.docs-page {
  display: grid;
  gap: 16px;
}

.docs-head,
.doc-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.docs-head {
  align-items: center;
}

.doc-section-head.compact {
  margin-top: 20px;
}

.essential-item,
.doc-card,
.code-panel {
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
  box-shadow: var(--app-shadow-sm);
}

.essential-item {
  display: grid;
  gap: 6px;
  min-height: 112px;
  padding: 18px;
}

.essential-item span,
.essential-item small,
.doc-section-head p,
.security-summary p,
.security-summary li,
.boundary-list p,
.step-item p,
.role-item p,
.crypto-step p,
.formula-list p,
.action-item p,
.action-item dd,
.trouble-item p,
.trouble-item small,
.note-list {
  color: var(--app-muted);
}

.essential-item strong {
  color: var(--app-text-strong);
  font-size: 20px;
  line-height: 28px;
}

.docs-tabs {
  min-width: 0;
}

.doc-card {
  overflow: hidden;
}

.doc-section-head h3 {
  margin: 0 0 6px;
  color: var(--app-text-strong);
  font-size: 18px;
  line-height: 26px;
}

.doc-section-head p {
  margin: 0;
  line-height: 22px;
}

.security-summary {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(280px, 0.9fr);
  gap: 16px;
  margin-bottom: 20px;
  padding: 18px;
  border: 1px solid color-mix(in srgb, var(--app-primary) 24%, var(--app-border));
  border-radius: 8px;
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--app-primary) 13%, transparent), transparent 58%),
    var(--app-surface-2);
}

.security-summary span {
  color: var(--app-primary);
  font-size: 12px;
  font-weight: 800;
}

.security-summary h3 {
  margin: 8px 0;
  color: var(--app-text-strong);
  font-size: 20px;
  line-height: 28px;
}

.security-summary p {
  margin: 0;
  line-height: 24px;
}

.security-summary ul {
  display: grid;
  gap: 10px;
  margin: 0;
  padding-left: 18px;
  line-height: 22px;
}

.role-grid,
.rule-grid,
.warning-grid,
.formula-list,
.code-grid,
.case-grid,
.action-item dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.role-grid {
  margin-top: 18px;
}

.role-item,
.crypto-step,
.step-item,
.formula-list div,
.action-item,
.trouble-item {
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.role-item {
  padding: 14px;
}

.role-item span,
.code-panel header span {
  display: block;
  color: var(--app-primary);
  font-size: 12px;
  font-weight: 800;
}

.role-item strong,
.crypto-step strong,
.step-item strong,
.formula-list strong,
.action-title strong,
.trouble-item strong {
  color: var(--app-text-strong);
}

.role-item p,
.crypto-step p,
.step-item p,
.formula-list p,
.action-item p {
  margin: 6px 0 0;
  line-height: 22px;
}

.crypto-flow,
.step-list,
.action-list,
.trouble-list,
.note-list {
  display: grid;
  gap: 12px;
}

.crypto-flow {
  margin-top: 12px;
}

.crypto-step,
.step-item {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 14px;
  padding: 14px;
}

.crypto-step span,
.step-item span {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: 8px;
  background: var(--app-primary-soft);
  color: var(--app-primary);
  font-weight: 800;
}

.warning-grid {
  margin-top: 16px;
}

.warning-grid p {
  margin: 6px 0 0;
}

.flow-diagram {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-top: 18px;
}

.flow-node {
  position: relative;
  display: grid;
  gap: 10px;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.flow-node::before {
  content: "";
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  border-radius: 8px 0 0 8px;
  background: var(--app-primary);
}

.flow-node:not(:nth-child(4n))::after {
  content: "→";
  position: absolute;
  z-index: 2;
  top: 44px;
  right: -14px;
  color: var(--app-primary);
  font-size: 18px;
  font-weight: 900;
}

.flow-node.lane-client::before {
  background: #20c997;
}

.flow-node.lane-network::before {
  background: #f59e0b;
}

.flow-node-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.flow-node-top > span {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 8px;
  background: var(--app-primary-soft);
  color: var(--app-primary);
  font-weight: 900;
}

.flow-node strong,
.key-map-row strong,
.request-step strong,
.flow-callout strong {
  color: var(--app-text-strong);
}

.flow-node p,
.key-map-row dd,
.request-step p,
.flow-callout p {
  margin: 0;
  color: var(--app-muted);
  line-height: 22px;
}

.flow-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.flow-chips small {
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--app-surface);
  color: var(--app-muted);
  font-size: 11px;
  line-height: 16px;
}

.key-map {
  display: grid;
  gap: 10px;
  margin-top: 14px;
}

.key-map-row {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.key-map-row > div:first-child {
  min-width: 0;
}

.key-map-row span {
  display: block;
  margin-bottom: 6px;
  color: var(--app-primary);
  font-size: 12px;
  font-weight: 800;
}

.key-map-row strong {
  overflow-wrap: anywhere;
  font-family: Consolas, "SFMono-Regular", monospace;
  font-size: 13px;
}

.key-map-row dl {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.key-map-row dt {
  margin-bottom: 4px;
  color: var(--app-text-strong);
  font-size: 12px;
  font-weight: 800;
}

.key-map-row dd {
  font-size: 12px;
  overflow-wrap: anywhere;
}

.request-rail {
  display: grid;
  gap: 10px;
  margin-top: 14px;
  padding-left: 17px;
  border-left: 2px solid color-mix(in srgb, var(--app-primary) 45%, var(--app-border));
}

.request-step {
  position: relative;
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.request-step::before {
  content: "";
  position: absolute;
  top: 30px;
  left: -24px;
  width: 12px;
  height: 12px;
  border: 3px solid var(--app-primary);
  border-radius: 50%;
  background: var(--app-surface);
}

.request-step > span {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 8px;
  background: var(--app-primary-soft);
  color: var(--app-primary);
  font-size: 12px;
  font-weight: 900;
}

.request-step p {
  margin-top: 6px;
}

.request-step code {
  display: block;
  margin-top: 8px;
  overflow-wrap: anywhere;
  color: var(--app-primary);
  font-family: Consolas, "SFMono-Regular", monospace;
  font-size: 12px;
  line-height: 20px;
}

.flow-callout-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.flow-callout {
  padding: 14px;
  border: 1px solid color-mix(in srgb, #f59e0b 38%, var(--app-border));
  border-radius: 8px;
  background: color-mix(in srgb, #f59e0b 10%, var(--app-surface-2));
}

.flow-callout p {
  margin-top: 8px;
}

.endpoint-list {
  display: grid;
  gap: 10px;
  margin: 18px 0;
}

.endpoint-list div {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.endpoint-list code,
.single-code,
.code-panel pre,
.formula-list pre {
  font-family: Consolas, "SFMono-Regular", monospace;
}

.endpoint-list code {
  color: var(--app-primary);
  font-size: 13px;
}

.formula-list div {
  padding: 14px;
}

.formula-list pre {
  margin: 10px 0;
  overflow: auto;
  padding: 12px;
  border-radius: 8px;
  background: var(--app-surface);
  color: var(--app-text);
  font-size: 12px;
  line-height: 20px;
}

.code-grid,
.case-grid {
  margin-top: 16px;
}

.code-panel {
  min-width: 0;
  overflow: hidden;
}

.code-panel header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border-bottom: 1px solid var(--app-border);
}

.code-panel header strong {
  color: var(--app-text-strong);
  font-size: 14px;
}

.code-panel pre,
.single-code {
  max-width: 100%;
  margin: 0;
  overflow: auto;
  padding: 14px;
  background: var(--app-surface-2);
  color: var(--app-text);
  font-size: 12px;
  line-height: 20px;
  white-space: pre;
}

.single-code {
  margin-top: 16px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.action-item,
.trouble-item {
  padding: 14px;
}

.action-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.action-item dl {
  margin: 12px 0 0;
}

.action-item dt {
  margin-bottom: 4px;
  color: var(--app-text-strong);
  font-weight: 700;
}

.action-item dd {
  margin: 0;
  line-height: 22px;
  word-break: break-word;
}

.trouble-item p {
  margin: 8px 0;
  line-height: 22px;
}

.trouble-item small {
  display: block;
  line-height: 20px;
}

.doc-alert {
  margin-top: 0;
}

.boundary-list {
  display: grid;
  gap: 12px;
  margin-bottom: 16px;
}

.boundary-list div {
  padding: 14px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-2);
}

.boundary-list strong {
  color: var(--app-text-strong);
}

.boundary-list p {
  margin: 8px 0 0;
  line-height: 22px;
}

.note-list {
  margin: 0;
  padding-left: 18px;
  line-height: 24px;
}

@media (max-width: 900px) {
  .security-summary,
  .role-grid,
  .rule-grid,
  .warning-grid,
  .formula-list,
  .code-grid,
  .case-grid,
  .action-item dl,
  .flow-diagram,
  .flow-callout-grid {
    grid-template-columns: 1fr;
  }

  .flow-node:not(:last-child)::after {
    content: "";
    top: auto;
    right: auto;
    bottom: -13px;
    left: 28px;
    width: 2px;
    height: 12px;
    background: var(--app-primary);
  }

  .key-map-row,
  .key-map-row dl {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .docs-head,
  .doc-section-head {
    display: grid;
  }

  .docs-head .n-button,
  .doc-section-head .n-button {
    width: 100%;
  }

  .crypto-step,
  .step-item,
  .request-step {
    grid-template-columns: 1fr;
  }

  .code-panel header {
    align-items: stretch;
    flex-direction: column;
  }

  .code-panel header .n-button {
    width: 100%;
  }
}
</style>
