<template>
  <div class="terminal-wrapper">
    <div class="terminal-toolbar">
      <div class="toolbar-title">
        <span class="dot" :class="statusClass"></span>
        <span class="name">{{ title }}</span>
        <div v-if="port > 0" class="port-url-badges">
          <a 
            :href="'http://127.0.0.1:' + port" 
            target="_blank" 
            class="port-url-badge" 
            title="在浏览器中打开: http://127.0.0.1:{{ port }}"
          >
            🌐 http://127.0.0.1:{{ port }} ↗
          </a>
          <a 
            :href="'http://localhost:' + port" 
            target="_blank" 
            class="port-url-badge" 
            title="在浏览器中打开: http://localhost:{{ port }}"
          >
            🌐 http://localhost:{{ port }} ↗
          </a>
          <a 
            v-if="localIp && localIp !== '127.0.0.1'" 
            :href="'http://' + localIp + ':' + port" 
            target="_blank" 
            class="port-url-badge" 
            title="在浏览器中打开局域网地址: http://{{ localIp }}:{{ port }}"
          >
            🌐 http://{{ localIp }}:{{ port }} ↗
          </a>
        </div>
      </div>
      <div class="toolbar-actions">
        <button class="tool-btn" @click="clearLogs" title="清屏">清屏</button>
        <button class="tool-btn" :class="{ active: autoScroll }" @click="autoScroll = !autoScroll" title="自动滚底">
          滚底: {{ autoScroll ? '开' : '关' }}
        </button>
      </div>
    </div>
    <div ref="terminalContainer" class="terminal-body"></div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch, computed } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'

const props = defineProps({
  serviceId: { type: String, required: true },
  title: { type: String, default: '' },
  port: { type: Number, default: 0 },
  localIp: { type: String, default: '' },
  status: { type: Number, default: 0 } // 0: stopped, 1: starting, 2: running
})


const terminalContainer = ref(null)
const autoScroll = ref(true)
let term = null
let fitAddon = null

const statusClass = computed(() => {
  if (props.status === 2) return 'dot-running glow-green'
  if (props.status === 1) return 'dot-starting glow-yellow'
  return 'dot-stopped'
})

const openBrowser = () => {
  if (props.port > 0) {
    window.open(`http://localhost:${props.port}`, '_blank')
  }
}

const clearLogs = () => {
  if (term) term.clear()
}

const appendLogs = (batch) => {
  if (!term || !batch || batch.length === 0) return
  const text = batch.join('\r\n') + '\r\n'
  term.write(text)
  if (autoScroll.value) {
    term.scrollToBottom()
  }
}

defineExpose({
  appendLogs,
  clearLogs
})

onMounted(() => {
  term = new Terminal({
    theme: {
      background: '#0d1017',
      foreground: '#e6edf3',
      cursor: '#58a6ff',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d4dd',
      brightWhite: '#ffffff',
    },
    fontSize: 12,
    fontFamily: 'Consolas, "Fira Code", Monaco, monospace',
    lineHeight: 1.3,
    convertEol: true,
    disableStdin: true,
    scrollback: 3000
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)

  // 注册终端超链接点击跳转提供器
  if (term.registerLinkProvider) {
    term.registerLinkProvider({
      provideLinks(bufferLineNumber, callback) {
        const line = term.buffer.active.getLine(bufferLineNumber - 1)
        if (!line) return callback(undefined)
        const lineText = line.translateToString(true)
        const urlRegex = /(https?:\/\/[^\s\x1b"'()<>]+)/g
        const links = []
        let match

        // 字符下标转终端 Cell 列网格物理坐标 (1-based)
        const getCellCol = (charIndex) => {
          let curCharIdx = 0
          let col = 0
          while (col < line.length && curCharIdx < charIndex) {
            const cell = line.getCell(col)
            if (!cell) break
            const width = cell.getWidth()
            const chars = cell.getChars()
            if (width === 0) {
              col++
              continue
            }
            curCharIdx += (chars ? chars.length : 1)
            col += width
          }
          return col + 1
        }

        while ((match = urlRegex.exec(lineText)) !== null) {
          const url = match[1]
          const startCol = getCellCol(match.index)
          const endCol = startCol + url.length
          links.push({
            range: {
              start: { x: startCol, y: bufferLineNumber },
              end: { x: endCol, y: bufferLineNumber }
            },
            text: url,
            activate(event, text) {
              window.open(text, '_blank')
            }
          })
        }
        callback(links.length > 0 ? links : undefined)
      }
    })
  }


  if (terminalContainer.value) {
    term.open(terminalContainer.value)
    setTimeout(() => {
      try { fitAddon.fit() } catch(e) {}
    }, 50)
  }


  let resizeObserver = null
  if (window.ResizeObserver && terminalContainer.value) {
    resizeObserver = new ResizeObserver(() => {
      try { fitAddon?.fit() } catch(e) {}
    })
    resizeObserver.observe(terminalContainer.value)
  }

  // 初始加载历史日志
  fetch(`/api/service/logs?id=${props.serviceId}`)
    .then(res => res.json())
    .then(logs => {
      if (logs && logs.length > 0 && term) {
        term.clear()
        appendLogs(logs)
      }
    })
    .catch(() => {})

})

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  if (term) {
    term.dispose()
    term = null
  }
})
</script>


<style scoped>
.terminal-wrapper {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  background: #0d1017;
  border-radius: 6px;
  overflow: hidden;
}
.terminal-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 12px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
}
.toolbar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 500;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.dot-running { background: #10b981; }
.dot-starting { background: #f59e0b; }
.dot-stopped { background: #6b7280; }
.port-url-badges {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.port-url-badge {
  color: #38bdf8;

  background: rgba(56, 189, 248, 0.12);
  border: 1px solid rgba(56, 189, 248, 0.3);
  padding: 2px 8px;
  border-radius: 4px;
  cursor: pointer;
  font-family: monospace;
  font-size: 11px;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: all 0.15s ease;
}
.port-url-badge:hover {
  background: rgba(56, 189, 248, 0.25);
  color: #fff;
  border-color: #38bdf8;
  transform: translateY(-1px);
}

.toolbar-actions {
  display: flex;
  gap: 6px;
}
.tool-btn {
  background: #21262d;
  color: #c9d1d9;
  border: 1px solid #30363d;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
}
.tool-btn:hover { background: #30363d; }
.tool-btn.active { color: #58a6ff; border-color: #58a6ff; }
.terminal-body {
  flex: 1;
  width: 100%;
  padding: 4px 6px;
  overflow: hidden;
}
</style>

