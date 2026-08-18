<template>
  <div class="rules-container">
    <!-- 顶部工具栏 -->
    <div class="rules-header">
      <div class="rules-search-group">
        <input 
          v-model="searchQuery" 
          type="text" 
          class="search-input" 
          placeholder="🔍 搜索规则名称、特征文件、启动命令..."
        />
        <div class="category-filters">
          <button 
            v-for="cat in categories" 
            :key="cat"
            class="cat-pill" 
            :class="{ active: selectedCategory === cat }"
            @click="selectedCategory = cat"
          >
            {{ cat }} ({{ getCategoryCount(cat) }})
          </button>
        </div>
      </div>
      <div class="rules-actions">
        <button class="btn btn-outline" @click="resetToDefault" title="重置为出厂预置规则">🔄 恢复默认</button>
        <button class="btn btn-primary" @click="openAddModal">➕ 新建识别规则</button>
      </div>
    </div>

    <!-- 规则表格 -->
    <div class="rules-table-wrapper">
      <table class="rules-table">
        <thead>
          <tr>
            <th>状态</th>
            <th>规则名称</th>
            <th>技术生态</th>
            <th>特征文件</th>
            <th>内容匹配</th>
            <th>启动命令</th>
            <th>默认端口</th>
            <th style="text-align: right;">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="rule in filteredRules" :key="rule.id" :class="{ 'row-disabled': !rule.enabled }">
            <td>
              <label class="switch">
                <input type="checkbox" :checked="rule.enabled" @change="toggleRule(rule)" />
                <span class="slider"></span>
              </label>
            </td>
            <td>
              <div class="rule-name-cell">
                <span class="rule-title">{{ rule.name }}</span>
                <span v-if="rule.isBuiltin" class="builtin-badge">预置</span>
              </div>
            </td>
            <td><span class="cat-badge">{{ rule.category }}</span></td>
            <td><code class="code-badge">{{ rule.matchFile }}</code></td>
            <td><code class="code-badge-sub" :title="rule.matchContent">{{ rule.matchContent || '任意' }}</code></td>
            <td><code class="code-badge-cmd">{{ rule.command }}</code></td>
            <td><span class="port-text">{{ rule.defaultPort > 0 ? ':' + rule.defaultPort : '无' }}</span></td>
            <td style="text-align: right;">
              <button class="icon-btn" @click="editRule(rule)" title="编辑">✏️</button>
              <button class="icon-btn btn-danger-hover" @click="deleteRule(rule.id)" title="删除">🗑️</button>
            </td>
          </tr>
          <tr v-if="filteredRules.length === 0">
            <td colspan="8" class="empty-rules">未找到匹配的项目识别规则</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 新建/编辑规则居中大弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-card modal-large">
        <div class="modal-header">
          <h3 class="modal-title">{{ isEditing ? '✏️ 编辑识别规则' : '➕ 新建识别规则' }}</h3>
          <button class="close-btn" @click="showModal = false">×</button>
        </div>
        <div class="modal-body">
          <div class="form-row">
            <div class="form-group flex-1">
              <label>规则名称 <span class="required">*</span></label>
              <input v-model="form.name" type="text" class="form-input" placeholder="例: Vite 前端 (dev)" />
            </div>
            <div class="form-group flex-1">
              <label>技术生态 <span class="required">*</span></label>
              <input v-model="form.category" type="text" class="form-input" placeholder="例: Node/前端, Python, .NET, Go" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-group flex-1">
              <label>特征匹配文件名 <span class="required">*</span></label>
              <input v-model="form.matchFile" type="text" class="form-input" placeholder="例: package.json, *.csproj, main.py" />
            </div>
            <div class="form-group flex-1">
              <label>文件内容关键字 (可选)</label>
              <input v-model="form.matchContent" type="text" class="form-input" placeholder="例: FastAPI, Microsoft.NET.Sdk.Web" />
            </div>
          </div>
          <div class="form-row">
            <div class="form-group flex-2">
              <label>默认启动命令模板 <span class="required">*</span></label>
              <input v-model="form.command" type="text" class="form-input" placeholder="例: pnpm dev, dotnet run, python main.py" />
            </div>
            <div class="form-group flex-1">
              <label>默认端口 (0为无端口)</label>
              <input v-model.number="form.defaultPort" type="number" class="form-input" placeholder="5173" />
            </div>
          </div>
          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="form.enabled" />
              <span>立即启用该识别规则</span>
            </label>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showModal = false">取消</button>
          <button class="btn btn-primary" @click="saveRule">保存规则</button>
        </div>
      </div>
    </div>
  </div>
</template>


<script setup>
import { ref, computed, onMounted } from 'vue'

const rules = ref([])
const searchQuery = ref('')
const selectedCategory = ref('全部')
const showModal = ref(false)
const isEditing = ref(false)

const form = ref({
  id: '',
  name: '',
  category: 'Node/前端',
  matchFile: '',
  matchContent: '',
  command: '',
  defaultPort: 0,
  portExtractRegex: '',
  enabled: true,
  isBuiltin: false
})

const categories = computed(() => {
  const set = new Set(['全部'])
  rules.value.forEach(r => { if (r.category) set.add(r.category) })
  return Array.from(set)
})

const getCategoryCount = (cat) => {
  if (cat === '全部') return rules.value.length
  return rules.value.filter(r => r.category === cat).length
}

const filteredRules = computed(() => {
  return rules.value.filter(r => {
    const matchCat = selectedCategory.value === '全部' || r.category === selectedCategory.value
    const q = searchQuery.value.trim().toLowerCase()
    const matchQuery = !q || 
      r.name?.toLowerCase().includes(q) || 
      r.matchFile?.toLowerCase().includes(q) || 
      r.command?.toLowerCase().includes(q) || 
      r.category?.toLowerCase().includes(q)
    return matchCat && matchQuery
  })
})

const fetchRules = async () => {
  try {
    const res = await fetch('/api/rules')
    rules.value = await res.json()
  } catch (e) {}
}

const saveRulesToServer = async () => {
  await fetch('/api/rules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(rules.value)
  })
}

const toggleRule = async (rule) => {
  rule.enabled = !rule.enabled
  await saveRulesToServer()
}

const openAddModal = () => {
  isEditing.value = false
  form.value = {
    id: 'rule-' + crypto.randomUUID().slice(0, 8),
    name: '',
    category: 'Node/前端',
    matchFile: '',
    matchContent: '',
    command: '',
    defaultPort: 0,
    portExtractRegex: '',
    enabled: true,
    isBuiltin: false
  }
  showModal.value = true
}

const editRule = (rule) => {
  isEditing.value = true
  form.value = JSON.parse(JSON.stringify(rule))
  showModal.value = true
}

const saveRule = async () => {
  if (!form.value.name || !form.value.matchFile || !form.value.command) {
    alert('请填写完整的规则名称、特征文件和启动命令')
    return
  }
  if (isEditing.value) {
    const idx = rules.value.findIndex(r => r.id === form.value.id)
    if (idx !== -1) rules.value[idx] = form.value
  } else {
    rules.value.push(form.value)
  }
  await saveRulesToServer()
  showModal.value = false
}

const deleteRule = async (id) => {
  if (confirm('确定删除该识别规则吗？')) {
    rules.value = rules.value.filter(r => r.id !== id)
    await saveRulesToServer()
  }
}

const resetToDefault = async () => {
  if (confirm('确定重置为出厂预置规则吗？自定义规则将会被重置。')) {
    const res = await fetch('/api/rules/reset', { method: 'POST' })
    rules.value = await res.json()
  }
}

onMounted(() => {
  fetchRules()
})
</script>

<style scoped>
.rules-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px 20px;
  background: var(--bg-main);
  overflow-y: auto;
  gap: 16px;
}
.rules-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.rules-search-group {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  flex: 1;
}
.search-input {
  width: 280px;
  background: #181d28;
  border: 1px solid var(--border-color);
  color: #fff;
  padding: 7px 12px;
  border-radius: 6px;
  font-size: 13px;
}
.category-filters { display: flex; gap: 6px; flex-wrap: wrap; }
.cat-pill {
  background: #1c2230;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 4px 10px;
  border-radius: 14px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.cat-pill:hover, .cat-pill.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}
.rules-actions { display: flex; gap: 8px; }
.rules-table-wrapper {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
}
.rules-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 13px;
}
.rules-table th {
  background: #141721;
  color: var(--text-secondary);
  font-weight: 500;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-color);
}
.rules-table td {
  padding: 10px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
}
.row-disabled td { opacity: 0.5; }
.rule-name-cell { display: flex; align-items: center; gap: 6px; }
.rule-title { font-weight: 600; color: #fff; }
.builtin-badge { font-size: 10px; background: rgba(59, 130, 246, 0.2); color: #60a5fa; padding: 1px 5px; border-radius: 4px; }
.cat-badge { font-size: 11px; background: #222938; padding: 2px 6px; border-radius: 4px; color: #cbd5e1; }
.code-badge { font-family: monospace; background: #0f131a; color: #38bdf8; padding: 2px 6px; border-radius: 4px; font-size: 12px; }
.code-badge-sub { font-family: monospace; background: #0f131a; color: #a78bfa; padding: 2px 6px; border-radius: 4px; font-size: 11px; }
.code-badge-cmd { font-family: monospace; background: #0f131a; color: #34d399; padding: 2px 6px; border-radius: 4px; font-size: 12px; }
.port-text { font-family: monospace; font-weight: 600; color: #fbbf24; }
.empty-rules { text-align: center; padding: 30px; color: var(--text-muted); }

/* Switch 开关样式 */
.switch { position: relative; display: inline-block; width: 34px; height: 18px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider {
  position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0;
  background-color: #374151; transition: .3s; border-radius: 18px;
}
.slider:before {
  position: absolute; content: ""; height: 14px; width: 14px; left: 2px; bottom: 2px;
  background-color: white; transition: .3s; border-radius: 50%;
}
input:checked + .slider { background-color: #10b981; }
input:checked + .slider:before { transform: translateX(16px); }

/* 居中大弹窗样式 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(5px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-card.modal-large {
  width: 680px;
  max-width: 92vw;
  background: #141721;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.7);
  display: flex;
  flex-direction: column;
}
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}
.modal-title { font-size: 15px; font-weight: 600; color: #fff; }
.close-btn {
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: 20px;
  cursor: pointer;
  line-height: 1;
}
.close-btn:hover { color: #fff; }
.modal-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: 75vh;
  overflow-y: auto;
}
.modal-footer {
  padding: 14px 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.form-row { display: flex; gap: 14px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 12px; color: var(--text-secondary); font-weight: 500; }
.required { color: #f87171; }
.form-input {
  background: #0d1017;
  border: 1px solid #2d3348;
  border-radius: 6px;
  padding: 8px 12px;
  color: #fff;
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
}
.form-input:focus { border-color: #3b82f6; }
.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-primary);
  cursor: pointer;
  margin-top: 4px;
}
.flex-1 { flex: 1; }
.flex-2 { flex: 2; }
</style>



