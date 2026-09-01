# 车机版 UI 真机渲染问题的验证方法 —— 以按钮图标居中修复为例

> 背景 issue：用户反馈车机版播放页圆形按钮内的图标、封面占位音符不在容器中央。
> 修复 commit：`f9b4b8490d37e2ba718ccb109f7d2bd5884b7920`（fix(car): 播放控制按钮图标与封面占位音符精确居中）
> 用户设备：OPPO Find N3 外屏浏览器（本地无此环境）
> 文档版本：v1.1
> 更新时间：2026-09-01（v1.1：修正「偏右」表述与 115% 机制归因，补充容器级缩放复核数据）

---

## 一、问题与挑战

用户在 OPPO Find N3 手机浏览器上访问车机版（`web/car/player.html`），反馈：

- 播放/上一首/下一首等圆形按钮内的 FontAwesome 图标明显偏下；
- 封面占位音符明显偏下，非常显眼。

  注（v1.1 修正）：像素实测音符水平方向居中（dx≈0）。FA 音符字形自身不对称
  （符杆偏右），视觉上易读作「偏右」，并非布局偏差。

挑战在于：**本地用 Chrome DevTools 模拟手机时偏差远小于用户截图**，且我们没有同款设备，无法直接确认修复是否在真机上生效。

## 二、根因分析

### 2.1 CSS 层面的根因（修复所解决的）

旧 CSS 用 `line-height` + `vertical-align: middle` 做垂直居中。FontAwesome 字体自带
`line-height: 1`，其字体度量（ascent/descent）与父级 strut 不一致，图标整体下沉；
且 ≤480px 断点下封面容器缩到 120px 而 `line-height` 仍是 150px，音符偏下更严重。

修复改为绝对定位几何居中，只依赖容器尺寸、与字体度量无关：

```css
.player-control-btn i {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
}
```

### 2.2 为什么本地模拟看不出那么大偏差（真机参数反推）

对用户截图（1116×2484，正好是 Find N3 外屏物理分辨率）做像素级分析，反推出用户浏览器的真实参数，与 Chrome 模拟器默认值差异很大：

| 参数 | Chrome 模拟器常用值 | 用户真机实际值 | 反推依据 |
|---|---|---|---|
| devicePixelRatio | 2.625 ~ 2.75 | **3.25** | 48px 按钮 ×3.25 = 156 物理px，与截图吻合 |
| CSS 视口宽度 | ~412px | **≈343px** | 1116 ÷ 3.25，命中 ≤480px 断点 |
| 字体缩放 | 100% | **≈115%**（ColorOS 字体放大） | 真机图标 ink 像素数比标准渲染多 ~30% |

关键是最后一项：**系统文字缩放会连同容器字号一起放大**（strut 随之变化，基线在固定
line-height 内下移），所以开了系统字体放大的用户看到的偏差比桌面模拟大得多。注意
放大的必须是容器字号——只放大图标 `i` 自身的 `font-size` 并不会加深偏移：
`vertical-align: middle` 的图标中心点不随字形大小移动，基线对齐的封面音符 bbox
中心反而上移（实测见第三节复核表）。而 FontAwesome 由项目本地打包
（`web/car/lib/fontawesome/`），排除了"字体没加载、fallback 字体度量不同"这个变量。

## 三、验证方法：让无头浏览器成为真机的像素级代理

思路分三步，每一步都有可量化的判据：

1. **量化真机截图（修复前）**：检测截图中的紫色圆钮/封面连通块，计算白色图标 ink
   的 bbox 中心相对容器 bbox 中心的偏移（dx/dy）。
2. **校准无头环境**：Playwright + Chromium 按反推参数（视口 343px、DPR 3.25）渲染
   修复前的页面，用同一算法测量。**校准判据：图标 ink 像素数与真机截图误差 <2%**
   （叠加 115% 字体缩放后，实测 1155 vs 1155、3543 vs 3582 等，完全吻合）——代理成立。
3. **同环境测修复后**：代理成立的前提下，修复后偏差归零即可断定真机上有效；再叠加
   `font-size: 115% !important` 做字体缩放鲁棒性检查。

### 实测数据（物理像素，正值 = 偏下）

| 元素 | 真机截图（修复前） | 无头·修复前 | 无头·修复后 | 修复后 + 字体放大115% |
|---|---|---|---|---|
| 封面音符（390px 容器） | +77 | +56.5 | **-1.5** | -1.0 |
| 小控制钮（156px） | +19 | +11 | **+1** | -4 |
| 播放大钮（194px） | +23 | +13 | **0** | -0.5 |

结论：修复后即使叠加字体放大，最大残差 4 物理px（≈1.2 CSS px），肉眼不可见。
居中只依赖容器几何尺寸，FontAwesome 字形在 em box 内的位置由字体文件本身决定、
各设备一致，因此没有理由在 OPPO 浏览器上失效。

### 独立复核补充（v1.1，PIL 像素管线，343×764 DPR 3.25）

对上表缺格与机制归因的补充实测（物理 px，正值 = 偏下）：

| 条件 | 小钮 dy | 大钮 dy | 封面 dy |
|---|---|---|---|
| 旧 CSS，无缩放 | +11.5 | +13 | +59.5 |
| 旧 CSS + 仅图标字号 ×1.15 | +13 | +15.5 | **+48（反而变小）** |
| 旧 CSS + 容器字号 ×1.15（模拟系统文字缩放） | +13 | +17.5 | **+62** |
| 新 CSS + 容器字号 ×1.15 | 0.0 | ≤+1.0 | 0.0 |
| 真机截图（对照） | +19 | +23 | +77 |

容器级缩放让偏差明显加深、逼近真机数值，证实「系统文字缩放放大容器字号」才是真机
偏差大于桌面模拟的主因；残余差距（约 2~5 CSS px）可能是 ColorOS 实际档位高于 115%
或厂商浏览器 text autosizing 实现差异。无论何种条件，**新 CSS 偏差均 ≤1 物理px**，
修复对文字缩放鲁棒的结论不变。

## 四、如何复用这套验证流程

### 4.1 环境

- Chromium：复用 Playwright 缓存 `~/.cache/ms-playwright/chromium-*/chrome-linux64/chrome`；
- playwright-core：可复用任意装过 Playwright 的项目（通过 `NODE_PATH` 指过去），
  无需在本仓库安装依赖。

### 4.2 运行

```bash
# 分析一张真机截图（输出各紫色组件内白色图标的偏移）
NODE_PATH=<playwright-core所在node_modules> node measure-icons.cjs shot <截图.png>

# 无头渲染并测量：old = 修复前 CSS（从 git 取），new = 工作区当前 CSS，
# new-scaled = 当前 CSS + 模拟 115% 字体放大
NODE_PATH=... node measure-icons.cjs live old|new|new-scaled
```

脚本要点（全文见附录）：

- `page.route('**/api/**', abort)` 屏蔽后端，让页面停留在与用户截图一致的占位态；
  API 失败会弹"选择身份"模态框，截图前需移除 `.modal / .modal-backdrop`，否则遮罩
  压暗颜色导致检测失败；
- 测"修复前"不需要切分支：`git show <commit>^:web/car/css/simple.css` 取旧文件，
  用 `page.route` 拦截 CSS 请求直接替换响应体；
- 像素分析在页面内 canvas 完成（截图转 dataURL 传入 `page.evaluate`），无需任何
  Node 图像库；
- `new-scaled` 只放大图标字号，仅能回归「图标自身尺寸变化不影响新方案居中」；
  若要模拟 Android 系统文字缩放，应缩放容器字号（桌面 Chromium 无 Android 专属的
  CDP `Emulation.setTextScaleFactor`），两种方式的差异见第三节复核表。

### 4.3 后续迭代的最终闭环手段

1. **最简单**：部署后请用户再发一张截图，跑一遍 `shot` 模式，数字归零即闭环。
   （用户微信反馈截图存放在 nuc 机器，`scp nuc:<路径> /tmp/` 拉取。）
2. **远程测量**：可给页面加 `?debug=1` 开关，用 JS 把每个按钮与图标的
   `getBoundingClientRect` 中心差、`devicePixelRatio`、视口尺寸直接渲染在页面上，
   用户截一张图即等于一次远程测量。
3. **真机矩阵**：BrowserStack 等云真机平台有 OPPO 设备，适用于更复杂的兼容性问题。

## 五、经验总结

- 手机浏览器渲染偏差报告中，**先从截图反推 DPR / 视口 / 字体缩放**，再谈复现——
  Chrome 模拟器默认参数与真机（尤其开了显示/字体放大的设备）可能差一个断点；
- 判断无头环境是否是真机的有效代理，**比对字形 ink 像素数**是简单可靠的判据；
- 涉及字体度量的居中问题，修复方向应选择**与字体度量无关的几何定位**
  （绝对定位 + translate，或 flex 居中），并用字体缩放做鲁棒性回归。

---

## 附录：measure-icons.cjs 全文

```javascript
/**
 * 测量车机播放页按钮图标/封面音符相对容器中心的偏移。
 * 模式1: node measure-icons.cjs live <old|new|new-scaled> —— 无头浏览器渲染并测量
 * 模式2: node measure-icons.cjs shot <png路径> —— 分析一张真机截图
 */
const path = require('path');
const { chromium } = require('playwright-core');

const REPO = '/home/ety001/workspace/multitune';
const EXE = process.env.HOME + '/.cache/ms-playwright/chromium-1228/chrome-linux64/chrome';
// 修复 commit；live old 模式会取它的父提交的 CSS
const FIX_COMMIT = 'f9b4b8490d37e2ba718ccb109f7d2bd5884b7920';

// 页面内像素分析函数（通过 evaluate 注入）：
// 给定 dataURL，找出紫色连通块（按钮/封面）和其中的白色 ink，计算偏移
const analyzeFn = `async (dataUrl) => {
  const img = new Image();
  await new Promise((res, rej) => { img.onload = res; img.onerror = rej; img.src = dataUrl; });
  const W = img.naturalWidth, H = img.naturalHeight;
  const cv = document.createElement('canvas'); cv.width = W; cv.height = H;
  const ctx = cv.getContext('2d', { willReadFrequently: true });
  ctx.drawImage(img, 0, 0);
  const d = ctx.getImageData(0, 0, W, H).data;
  const isPurple = (i) => {
    const r = d[i], g = d[i+1], b = d[i+2];
    return b > 190 && r > 60 && r < 180 && g > 60 && g < 180 && b - r > 40;
  };
  const isWhite = (i) => d[i] > 200 && d[i+1] > 200 && d[i+2] > 200;
  // 连通块标记（紫色或白色都归入块，这样图标 ink 属于按钮块内部）
  const seen = new Uint8Array(W * H);
  const comps = [];
  for (let y = 0; y < H; y++) {
    for (let x = 0; x < W; x++) {
      const p = y * W + x;
      if (seen[p] || !isPurple(p * 4)) continue;
      const stack = [p];
      seen[p] = 1;
      let minX = x, maxX = x, minY = y, maxY = y, n = 0;
      let wMinX = 1e9, wMaxX = -1, wMinY = 1e9, wMaxY = -1, wCx = 0, wCy = 0, wN = 0;
      while (stack.length) {
        const q = stack.pop();
        const qx = q % W, qy = (q / W) | 0;
        n++;
        if (qx < minX) minX = qx; if (qx > maxX) maxX = qx;
        if (qy < minY) minY = qy; if (qy > maxY) maxY = qy;
        for (const [dx, dy] of [[1,0],[-1,0],[0,1],[0,-1]]) {
          const nx = qx + dx, ny = qy + dy;
          if (nx < 0 || ny < 0 || nx >= W || ny >= H) continue;
          const np = ny * W + nx;
          if (seen[np]) continue;
          const i4 = np * 4;
          if (isPurple(i4) || isWhite(i4)) {
            seen[np] = 1;
            stack.push(np);
            if (isWhite(i4)) {
              if (nx < wMinX) wMinX = nx; if (nx > wMaxX) wMaxX = nx;
              if (ny < wMinY) wMinY = ny; if (ny > wMaxY) wMaxY = ny;
              wCx += nx; wCy += ny; wN++;
            }
          }
        }
      }
      if (n > 400 && wN > 50) {
        comps.push({
          w: maxX - minX + 1, h: maxY - minY + 1,
          center: [(minX + maxX) / 2, (minY + maxY) / 2],
          inkCenter: [(wMinX + wMaxX) / 2, (wMinY + wMaxY) / 2],
          inkCentroid: [wCx / wN, wCy / wN],
          inkN: wN,
        });
      }
    }
  }
  comps.sort((a, b) => a.center[1] - b.center[1] || a.center[0] - b.center[0]);
  return comps.map(c => ({
    at: c.center.map(v => Math.round(v)),
    size: [c.w, c.h],
    dxBbox: +(c.inkCenter[0] - c.center[0]).toFixed(2),
    dyBbox: +(c.inkCenter[1] - c.center[1]).toFixed(2),
    dxCentroid: +(c.inkCentroid[0] - c.center[0]).toFixed(2),
    dyCentroid: +(c.inkCentroid[1] - c.center[1]).toFixed(2),
    inkN: c.inkN,
  }));
}`;

async function main() {
  const [mode, arg] = process.argv.slice(2);
  const browser = await chromium.launch({ executablePath: EXE });
  // OPPO Find N3 外屏实测参数（从用户截图反推）：
  // 物理 1116x2484，DPR 3.25 → CSS 视口约 343px 宽，命中 ≤480px 断点
  const context = await browser.newContext({
    viewport: { width: 343, height: 764 },
    deviceScaleFactor: 3.25,
    isMobile: true,
    hasTouch: true,
    userAgent: 'Mozilla/5.0 (Linux; Android 14; PHN110) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36',
  });
  const page = await context.newPage();

  if (mode === 'shot') {
    const fs = require('fs');
    const dataUrl = 'data:image/png;base64,' + fs.readFileSync(arg).toString('base64');
    await page.goto('about:blank');
    const result = await page.evaluate(`(${analyzeFn})(${JSON.stringify(dataUrl)})`);
    console.log(JSON.stringify({ mode: 'real-screenshot', file: arg, comps: result }, null, 1));
  } else {
    // live 模式: old = 修复前 CSS（从 git 取父提交版本）, new = 工作区当前 CSS
    if (arg === 'old') {
      const { execSync } = require('child_process');
      const oldCss = execSync(
        `git -C ${REPO} show ${FIX_COMMIT}^:web/car/css/simple.css`
      );
      await page.route('**/css/simple.css*', r =>
        r.fulfill({ body: oldCss, contentType: 'text/css' })
      );
    }
    if (arg === 'new-scaled') {
      // 模拟 ColorOS 字体放大 ~115%：图标字号整体放大后仍应居中
      await page.addInitScript(`document.addEventListener('DOMContentLoaded', () => {
        const st = document.createElement('style');
        st.textContent = '.player-control-btn i, .player-cover i { font-size: 115% !important; }';
        document.head.appendChild(st);
      })`);
    }
    // 屏蔽后端接口，让页面停留在占位状态（与用户截图一致的静态 UI）
    await page.route('**/api/**', r => r.abort());
    await page.goto('file://' + path.join(REPO, 'web/car/player.html'));
    await page.evaluate('document.fonts.ready.then(()=>1)');
    await page.waitForTimeout(600);
    // 关掉因 API 不可用而弹出的模态框及遮罩，恢复页面原始配色
    await page.evaluate(`(() => {
      document.querySelectorAll('.modal, .modal-backdrop, .toast').forEach(e => e.remove());
      document.body.classList.remove('modal-open');
      document.body.style.overflow = '';
    })()`);
    await page.waitForTimeout(200);
    const shot = await page.screenshot({ fullPage: false });
    const fs = require('fs');
    const outPng = `/tmp/car-headless-${arg}.png`;
    fs.writeFileSync(outPng, shot);
    const dataUrl = 'data:image/png;base64,' + shot.toString('base64');
    const result = await page.evaluate(`(${analyzeFn})(${JSON.stringify(dataUrl)})`);
    console.log(JSON.stringify({ mode: `headless-${arg}`, png: outPng, comps: result }, null, 1));
  }
  await browser.close();
}
main().catch(e => { console.error(e); process.exit(1); });
```
