<template>
  <div class="app-layout">
    <!-- 顶部导航栏 -->
    <header class="app-header">
      <div class="header-left">
        <span class="logo-icon">⚡</span>
        <h1 class="logo-title">AppsManager</h1>
        <span class="version-tag">v1.0</span>
        <div class="nav-tabs">

          <button 
            class="nav-tab-btn" 
            :class="{ active: currentView === 'monitor' }" 
            @click="currentView = 'monitor'"
          >🖥️ 服务监控</button>
          <button 
            class="nav-tab-btn" 
            :class="{ active: currentView === 'rules' }" 
            @click="currentView = 'rules'"
          >⚙️ 识别规则管理</button>
        </div>
      </div>
      <div class="header-center">
        <div v-if="currentView === 'monitor'" class="stats-pill">
          <span class="dot-running glow-green"></span>
          <span>运行中: {{ runningServicesCount }} / {{ totalServicesCount }}</span>
        </div>
      </div>
      <div class="header-right">
        <button v-if="currentView === 'monitor'" class="btn btn-secondary" @click="openNewProjectModal">
          <span>➕</span> 新建项目
        </button>
        <button v-if="currentView === 'monitor'" class="btn btn-warning" @click="stopAllServices" :disabled="runningServicesCount === 0">
          <span>⏹</span> 停止所有
        </button>
        <button class="btn btn-danger" @click="exitApplication" title="停止所有服务并完全退出应用程序">
          <span>🛑</span> 退出程序
        </button>
      </div>
    </header>

    <!-- 全局轻量 Toast 提示 -->
    <transition name="toast">
      <div v-if="toastMsg" class="global-toast">{{ toastMsg }}</div>
    </transition>


    <!-- 退出提示遮罩 -->
    <div v-if="isAppExited" class="modal-overlay">
      <div class="modal-card" style="width: 380px; text-align: center; padding: 24px;">
        <div style="font-size: 36px; margin-bottom: 12px;">👋</div>
        <h3 style="margin-bottom: 8px;">AppsManager 已安全退出</h3>
        <p style="color: var(--text-secondary); font-size: 13px; margin-bottom: 16px;">
          所有子服务已成功停止，后台进程已完全释放。
        </p>
        <button class="btn btn-primary" style="width: 100%; justify-content: center;" @click="closeWindow">
          关闭此页面
        </button>
      </div>
    </div>

    <!-- 识别规则管理页面视图 -->
    <div v-if="currentView === 'rules'" class="app-rules-view">
      <RulesManager />
    </div>

    <!-- 主工作区 (服务监控) -->
    <div v-else class="app-main">

      <!-- 左侧项目分组与导航 -->
      <aside class="app-sidebar">
        <div class="sidebar-header-row">
          <span class="sidebar-section-title">项目分组</span>
          <button class="sidebar-add-btn" @click="addGroup" title="新建分组">➕</button>
        </div>
        <div class="group-list">
          <!-- 固定项 1: 全部项目 -->
          <div 
            class="group-item" 
            :class="{ 
              active: selectedGroup === 'ALL',
              'drop-target-active': dropTargetGroup === 'ALL'
            }"
            @click="selectedGroup = 'ALL'"
            @dragover="onGroupDragOver($event, 'ALL')"
            @dragenter="onGroupDragEnter('ALL')"
            @dragleave="onGroupDragLeave('ALL')"
            @drop="onGroupDrop('ALL')"
          >
            <div class="group-left-content">
              <span class="group-name-text">全部项目</span>
            </div>
            <div class="group-right-wrap">
              <span class="group-badge">({{ projects.length }})</span>
            </div>
          </div>

          <!-- 固定项 2: 默认分组 -->
          <div 
            class="group-item" 
            :class="{ 
              active: selectedGroup === '默认',
              'drop-target-active': dropTargetGroup === '默认'
            }"
            @click="selectedGroup = '默认'"
            @dragover="onGroupDragOver($event, '默认')"
            @dragenter="onGroupDragEnter('默认')"
            @dragleave="onGroupDragLeave('默认')"
            @drop="onGroupDrop('默认')"
          >
            <div class="group-left-content">
              <span class="group-name-text">默认</span>
            </div>
            <div class="group-right-wrap">
              <span class="group-badge">({{ getGroupCount('默认') }})</span>
            </div>
          </div>

          <!-- 自定义可拖拽排序分组项 -->
          <div 
            v-for="(group, idx) in customGroupList" 
            :key="group"
            class="group-item custom-group-item"
            :class="{ 
              active: selectedGroup === group,
              'drop-target-active': dropTargetGroup === group,
              'reorder-target-active': groupReorderTargetIdx === idx,
              'group-dragging': draggedGroupIdx === idx
            }"
            draggable="true"
            @dragstart="onGroupDragStart($event, idx)"
            @dragover="onCustomGroupDragOver($event, group, idx)"
            @dragenter="onCustomGroupDragEnter(group, idx)"
            @dragleave="onCustomGroupDragLeave(group, idx)"
            @drop="onCustomGroupDrop(group, idx)"
            @dragend="onGroupDragEnd"
            @click="selectedGroup = group"
          >
            <div class="group-left-content">
              <span class="group-drag-handle" title="按住上下拖拽调整分组顺序">
                <svg width="10" height="14" viewBox="0 0 10 16" fill="currentColor">
                  <circle cx="2.5" cy="2.5" r="1.5"/>
                  <circle cx="7.5" cy="2.5" r="1.5"/>
                  <circle cx="2.5" cy="8" r="1.5"/>
                  <circle cx="7.5" cy="8" r="1.5"/>
                  <circle cx="2.5" cy="13.5" r="1.5"/>
                  <circle cx="7.5" cy="13.5" r="1.5"/>
                </svg>
              </span>
              <span class="group-name-text" :title="group">{{ group }}</span>
            </div>

            <div class="group-right-wrap">
              <div class="group-hover-btns">
                <button class="group-action-btn" @click.stop="renameGroup(group)" title="重命名分组">✏️</button>
                <button class="group-action-btn btn-danger-hover" @click.stop="deleteGroup(group)" title="删除分组">🗑️</button>
              </div>
              <span class="group-badge">({{ getGroupCount(group) }})</span>
            </div>
          </div>
        </div>
      </aside>

      <!-- 中间卡片网格与底部终端 -->
      <main class="app-content">
        <!-- 项目卡片网格 -->
        <div class="cards-container">
          <div 
            v-for="proj in filteredProjects" 
            :key="proj.Id" 
            class="project-card"
            :class="{ 
              'card-active': selectedProjectId === proj.Id,
              'card-is-dragging': draggedProject?.Id === proj.Id
            }"
            draggable="true"
            @dragstart="onCardDragStart($event, proj)"
            @dragend="onCardDragEnd"
            @click="selectProject(proj)"
          >

            <div class="card-header">
              <div class="card-title-group">
                <span class="project-name">{{ proj.Name }}</span>
                <button class="copy-svg-btn" @click.stop="copyText(proj.Name, '项目名称')" title="复制项目名称">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                  </svg>
                </button>
                <span v-if="proj.Group && proj.Group !== '默认' && proj.Group !== 'Default'" class="badge">{{ proj.Group }}</span>
              </div>
              <div class="card-actions">
                <button class="icon-btn" @click.stop="openFolder(proj.Path)" title="打开文件夹">📂</button>
                <button class="icon-btn" @click.stop="openVSCode(proj.Path)" title="在 VS Code 中打开">💻</button>
                <button class="icon-btn" @click.stop="editProject(proj)" title="编辑项目">✏️</button>
                <button class="icon-btn btn-danger-hover" @click.stop="deleteProject(proj.Id)" title="删除">🗑️</button>
              </div>


            </div>
            
            <div class="card-path-row">
              <span class="card-path" :title="proj.Path">{{ proj.Path }}</span>
              <button class="copy-svg-btn" @click.stop="copyText(proj.Path, '物理路径')" title="复制物理路径">
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
                </svg>
              </button>
            </div>



            <!-- 子服务列表 -->
            <div class="sub-services-list">
              <div 
                v-for="svc in proj.SubServices" 
                :key="svc.Id" 
                class="svc-row"
                :class="{ 'svc-row-active': activeTerminalId === svc.Id }"
                @click.stop="selectSubService(proj, svc)"
                title="点击切换到该服务控制台"
              >
                <div class="svc-info">
                  <span class="dot" :class="getSvcDotClass(svc)"></span>
                  <span class="svc-name">{{ svc.Name }}</span>
                  <a 
                    v-if="svc.Port > 0" 
                    :href="'http://127.0.0.1:' + svc.Port"
                    target="_blank"
                    class="svc-port-badge" 
                    :class="{ 'port-active': svc.Status === 2 }"
                    @click.stop
                    :title="svc.Status === 2 ? '运行中 - 点击打开 http://127.0.0.1:' + svc.Port : '端口 :' + svc.Port"
                  >
                    :{{ svc.Port }} ↗
                  </a>
                </div>

                <div class="svc-controls">
                  <button 
                    v-if="svc.Status !== 2 && svc.Status !== 1" 
                    class="action-btn btn-run"
                    @click.stop="startService(svc.Id)"
                  >▶ 启动</button>
                  <button 
                    v-else 
                    class="action-btn btn-stop"
                    @click.stop="stopService(svc.Id)"
                  >⏹ 停止</button>
                  <button class="action-btn btn-icon" @click.stop="restartService(svc.Id)" title="重启">↻</button>
                  <button 
                    v-if="svc.Port > 0" 
                    class="action-btn btn-icon btn-bolt" 
                    @click.stop="killPort(svc.Port, svc.Id)" 
                    title="一键强杀端口占用"
                  >⚡</button>
                </div>
              </div>
            </div>
          </div>
        </div>


        <!-- 底部终端日志抽屉 (支持上下拉动边框调整高度) -->
        <div 
          class="terminal-drawer" 
          :class="{ collapsed: isTerminalCollapsed }"
          :style="{ height: isTerminalCollapsed ? '36px' : terminalHeight + 'px' }"
        >
          <!-- 顶部拉伸调整手柄 -->
          <div 
            v-show="!isTerminalCollapsed" 
            class="drawer-resizer" 
            @mousedown="startResize"
            title="上下拖动调整终端高度"
          ></div>
          <div class="drawer-header">
            <div class="drawer-tabs">
              <div 
                v-for="tab in terminalTabs" 
                :key="tab.id" 
                class="tab-item"
                :class="{ active: activeTerminalId === tab.id }"
                @click="activeTerminalId = tab.id"
              >
                <span class="dot" :class="getTabDotClass(tab)"></span>
                <span>{{ tab.title }}</span>
                <span class="tab-close" @click.stop="closeTerminalTab(tab.id)">×</span>
              </div>
              <div v-if="terminalTabs.length === 0" class="empty-tabs">
                点击子服务 🖥️ 按钮展开控制台日志流
              </div>
            </div>
            <div class="drawer-controls">
              <button class="icon-btn" @click="isTerminalCollapsed = !isTerminalCollapsed">
                {{ isTerminalCollapsed ? '▲ 展开终端' : '▼ 折叠' }}
              </button>
            </div>
          </div>
          <div class="drawer-body" v-show="!isTerminalCollapsed && terminalTabs.length > 0">
            <div 
              v-for="tab in terminalTabs" 
              :key="tab.id" 
              class="terminal-pane"
              v-show="activeTerminalId === tab.id"
            >
              <TerminalTab 
                :ref="el => setTerminalRef(tab.id, el)"
                :service-id="tab.id"
                :title="tab.title"
                :port="tab.port"
                :local-ip="systemLocalIp"
                :status="tab.status"
              />
            </div>
          </div>
        </div>
      </main>
    </div>

    <!-- 新建/编辑项目弹窗 -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-card">
        <div class="modal-header">
          <h3>{{ isEditing ? '编辑项目' : '新建项目' }}</h3>
          <button class="icon-btn" @click="showModal = false">×</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>项目名称</label>
            <input v-model="form.Name" type="text" placeholder="例如: MyApp" class="form-input" />
          </div>
          <div class="form-group">
            <label>物理路径</label>
            <div class="input-with-btn">
              <input v-model="form.Path" type="text" placeholder="请选择或输入物理路径，如 D:\projects\MyApp" class="form-input" />
              <button class="btn btn-secondary" @click="selectFolder" title="打开系统对话框选择本地项目文件夹">
                <span>📁</span> 选择文件夹
              </button>
              <button class="btn btn-secondary" @click="autoDetectServices" title="扫描路径下的启动项">
                <span>🔍</span> 智能识别
              </button>
            </div>
          </div>
          <div class="form-group">
            <div class="sub-header">
              <label>分组分类</label>
              <button type="button" class="btn-sm btn-secondary" @click="promptNewGroupInModal">➕ 新建分组</button>
            </div>
            <div class="group-select-row">
              <select v-model="form.Group" class="form-input form-select">
                <option v-for="g in allGroups" :key="g" :value="g">{{ g }}</option>
              </select>
            </div>
          </div>


          <div class="form-group">
            <div class="sub-header">
              <label>子启动项列表</label>
              <button class="btn-sm btn-secondary" @click="addSubServiceRow">➕ 添加启动项</button>
            </div>
            <div class="sub-table">
              <div v-for="(svc, idx) in form.SubServices" :key="idx" class="sub-edit-row">
                <input v-model="svc.Name" placeholder="服务名 (如前端)" class="form-input flex-1" />
                <input v-model="svc.StartCommand" placeholder="启动命令 (如 pnpm dev)" class="form-input flex-2" />
                <input v-model.number="svc.Port" type="number" placeholder="端口" class="form-input flex-0-8" />
                <button class="icon-btn btn-danger-hover" @click="removeSubServiceRow(idx)">🗑️</button>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showModal = false">取消</button>
          <button class="btn btn-primary" @click="saveProject">保存项目</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import TerminalTab from './components/TerminalTab.vue'
import RulesManager from './components/RulesManager.vue'

const currentView = ref('monitor')
const projects = ref([])
const selectedGroup = ref('ALL')

const selectedProjectId = ref(null)
const isTerminalCollapsed = ref(false)
const terminalHeight = ref(Number(localStorage.getItem('terminalHeight')) || 280)
const terminalTabs = ref([])
const activeTerminalId = ref(null)
const terminalRefs = new Map()

// 终端上下拖拽拉伸高度逻辑
const startResize = (e) => {
  e.preventDefault()
  const startY = e.clientY
  const startHeight = terminalHeight.value

  const onMouseMove = (moveEvent) => {
    const deltaY = startY - moveEvent.clientY
    const minHeight = 100
    const maxHeight = window.innerHeight - 100
    const newHeight = Math.max(minHeight, Math.min(maxHeight, startHeight + deltaY))
    terminalHeight.value = newHeight
  }

  const onMouseUp = () => {
    localStorage.setItem('terminalHeight', String(terminalHeight.value))
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
  }

  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
}


const showModal = ref(false)
const isEditing = ref(false)
const form = ref({
  Id: '',
  Name: '',
  Path: '',
  Group: '默认',
  SubServices: []
})

let ws = null
const systemLocalIp = ref('')

const CUSTOM_GROUPS_KEY = 'appsmanager_custom_groups'
const customGroups = ref([])

try {
  const saved = localStorage.getItem(CUSTOM_GROUPS_KEY)
  if (saved) {
    customGroups.value = JSON.parse(saved)
  }
} catch (e) {}

const saveCustomGroups = () => {
  try {
    localStorage.setItem(CUSTOM_GROUPS_KEY, JSON.stringify(customGroups.value))
  } catch (e) {}
}

const customGroupList = computed(() => {
  const gSet = new Set(customGroups.value.filter(g => g && g !== '默认' && g !== 'ALL'))
  projects.value.forEach(p => {
    if (p.Group && p.Group !== '默认' && p.Group !== 'ALL') {
      gSet.add(p.Group)
    }
  })
  return Array.from(gSet)
})

const allGroups = computed(() => {
  const list = ['默认', ...customGroupList.value]
  if (form.value.Group && !list.includes(form.value.Group)) {
    list.push(form.value.Group)
  }
  return list
})

const getGroupCount = (group) => {
  return projects.value.filter(p => (p.Group || '默认') === group).length
}

// ---- 卡片拖拽移入分组逻辑 ----
const draggedProject = ref(null)
const isDraggingCard = ref(false)
const dropTargetGroup = ref(null)

const onCardDragStart = (e, proj) => {
  draggedProject.value = proj
  isDraggingCard.value = true
  e.dataTransfer.setData('source-type', 'card')
  e.dataTransfer.setData('project-id', proj.Id)
  e.dataTransfer.effectAllowed = 'move'
}

const onCardDragEnd = () => {
  draggedProject.value = null
  isDraggingCard.value = false
  dropTargetGroup.value = null
}

const onGroupDragOver = (e, group) => {
  if (isDraggingCard.value) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }
}

const onGroupDragEnter = (group) => {
  if (isDraggingCard.value) {
    dropTargetGroup.value = group
  }
}

const onGroupDragLeave = (group) => {
  if (dropTargetGroup.value === group) {
    dropTargetGroup.value = null
  }
}

const onGroupDrop = async (group) => {
  if (!isDraggingCard.value || !draggedProject.value) return
  const targetGroup = (group === 'ALL' || !group) ? '默认' : group
  const proj = draggedProject.value
  dropTargetGroup.value = null
  if (proj.Group === targetGroup) return

  proj.Group = targetGroup
  await fetch('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(projects.value)
  })
  showToast(`✅ 已将项目「${proj.Name}」移入分组「${targetGroup}」`)
  fetchProjects()
}

// ---- 自定义分组拖拽排序逻辑 ----
const draggedGroupIdx = ref(null)
const groupReorderTargetIdx = ref(null)

const onGroupDragStart = (e, index) => {
  if (isDraggingCard.value) return
  draggedGroupIdx.value = index
  e.dataTransfer.setData('source-type', 'group')
  e.dataTransfer.setData('group-index', index.toString())
  e.dataTransfer.effectAllowed = 'move'
}

const onCustomGroupDragOver = (e, group, idx) => {
  if (isDraggingCard.value) {
    onGroupDragOver(e, group)
    return
  }
  if (draggedGroupIdx.value !== null && draggedGroupIdx.value !== idx) {
    e.preventDefault()
    groupReorderTargetIdx.value = idx
    e.dataTransfer.dropEffect = 'move'
  }
}

const onCustomGroupDragEnter = (group, idx) => {
  if (isDraggingCard.value) {
    onGroupDragEnter(group)
    return
  }
  if (draggedGroupIdx.value !== null && draggedGroupIdx.value !== idx) {
    groupReorderTargetIdx.value = idx
  }
}

const onCustomGroupDragLeave = (group, idx) => {
  if (isDraggingCard.value) {
    onGroupDragLeave(group)
    return
  }
  if (groupReorderTargetIdx.value === idx) {
    groupReorderTargetIdx.value = null
  }
}

const onCustomGroupDrop = async (group, idx) => {
  if (isDraggingCard.value) {
    await onGroupDrop(group)
    return
  }
  if (draggedGroupIdx.value !== null && draggedGroupIdx.value !== idx) {
    const list = [...customGroupList.value]
    const [moved] = list.splice(draggedGroupIdx.value, 1)
    list.splice(idx, 0, moved)
    customGroups.value = list
    saveCustomGroups()
    draggedGroupIdx.value = null
    groupReorderTargetIdx.value = null
    showToast(`✅ 已调整分组顺序`)
  }
}

const onGroupDragEnd = () => {
  draggedGroupIdx.value = null
  groupReorderTargetIdx.value = null
}


const addGroup = () => {
  const name = prompt('请输入新分组名称:')
  if (!name || !name.trim()) return
  const cleanName = name.trim()
  if (!customGroups.value.includes(cleanName)) {
    customGroups.value.push(cleanName)
    saveCustomGroups()
  }
  selectedGroup.value = cleanName
  showToast(`✅ 已创建分组: ${cleanName}`)
}


const renameGroup = async (oldName) => {
  const newName = prompt(`请输入分组「${oldName}」的新名称:`, oldName)
  if (!newName || !newName.trim() || newName.trim() === oldName) return
  const cleanNewName = newName.trim()
  
  projects.value.forEach(p => {
    if (p.Group === oldName) p.Group = cleanNewName
  })
  const idx = customGroups.value.indexOf(oldName)
  if (idx !== -1) customGroups.value[idx] = cleanNewName
  if (selectedGroup.value === oldName) selectedGroup.value = cleanNewName
  saveCustomGroups()
  
  await fetch('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(projects.value)
  })
  showToast(`✅ 分组已重命名为: ${cleanNewName}`)
  fetchProjects()
}

const deleteGroup = async (groupName) => {
  if (groupName === '默认') {
    alert('默认分组不能删除')
    return
  }
  const count = getGroupCount(groupName)
  const tip = count > 0 
    ? `确定删除分组「${groupName}」吗？\n该分组下的 ${count} 个项目将自动移至「默认」分组。`
    : `确定删除分组「${groupName}」吗？`
  if (confirm(tip)) {
    projects.value.forEach(p => {
      if (p.Group === groupName) p.Group = '默认'
    })
    customGroups.value = customGroups.value.filter(g => g !== groupName)
    saveCustomGroups()
    if (selectedGroup.value === groupName) selectedGroup.value = 'ALL'
    
    await fetch('/api/projects', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(projects.value)
    })
    showToast(`🗑️ 已删除分组: ${groupName}`)
    fetchProjects()
  }
}


const filteredProjects = computed(() => {
  if (selectedGroup.value === 'ALL') return projects.value
  return projects.value.filter(p => (p.Group || '默认') === selectedGroup.value)
})


const totalServicesCount = computed(() => {
  return projects.value.reduce((acc, p) => acc + (p.SubServices?.length || 0), 0)
})

const runningServicesCount = computed(() => {
  return projects.value.reduce((acc, p) => {
    return acc + (p.SubServices?.filter(s => s.Status === 2)?.length || 0)
  }, 0)
})

const setTerminalRef = (id, el) => {
  if (el) terminalRefs.set(id, el)
  else terminalRefs.delete(id)
}

const getSvcDotClass = (svc) => {
  if (svc.Status === 2) return 'dot-running glow-green'
  if (svc.Status === 1) return 'dot-starting glow-yellow'
  return 'dot-stopped'
}

const getTabDotClass = (tab) => {
  if (tab.status === 2) return 'dot-running glow-green'
  if (tab.status === 1) return 'dot-starting glow-yellow'
  return 'dot-stopped'
}

const initWebSocket = () => {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws`
  ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'log_batch' && msg.serviceId) {
        const termComp = terminalRefs.get(msg.serviceId)
        if (termComp) {
          termComp.appendLogs(msg.data)
        }
      } else if (msg.type === 'status' && msg.serviceId) {
        // 更新内存中子服务的状态
        projects.value.forEach(p => {
          p.SubServices?.forEach(s => {
            if (s.Id === msg.serviceId) s.Status = msg.status
          })
        })
        terminalTabs.value.forEach(t => {
          if (t.id === msg.serviceId) t.status = msg.status
        })
      } else if (msg.type === 'port_corrected' && msg.serviceId) {
        projects.value.forEach(p => {
          p.SubServices?.forEach(s => {
            if (s.Id === msg.serviceId) s.Port = msg.data
          })
        })
        terminalTabs.value.forEach(t => {
          if (t.id === msg.serviceId) t.port = msg.data
        })
      } else if (msg.type === 'sync' && msg.data) {
        projects.value = msg.data
      }
    } catch (e) {}
  }

  ws.onclose = () => {
    setTimeout(initWebSocket, 2000)
  }
}

const fetchProjects = async () => {
  try {
    const res = await fetch('/api/projects')
    projects.value = await res.json()
  } catch (e) {}
}

const toastMsg = ref('')
let toastTimer = null
const showToast = (msg) => {
  toastMsg.value = msg
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toastMsg.value = ''
  }, 2200)
}

const copyText = (text, label = '内容') => {
  if (!text) return
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(() => {
      showToast(`✅ 已复制${label}: ${text}`)
    }).catch(() => {
      prompt(`请手动复制${label}:`, text)
    })
  } else {
    prompt(`请手动复制${label}:`, text)
  }
}




const startService = async (serviceId) => {
  await fetch('/api/service/start', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ serviceId })
  })
}

const stopService = async (serviceId) => {
  await fetch('/api/service/stop', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ serviceId })
  })
}

const restartService = async (serviceId) => {
  await fetch('/api/service/restart', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ serviceId })
  })
}

const killPort = async (port, serviceId) => {
  if (confirm(`确定强制终止占用端口 ${port} 的所有进程吗？`)) {
    await fetch('/api/service/kill_port', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ port, serviceId })
    })
  }
}

const selectProject = (proj) => {
  selectedProjectId.value = proj.Id
  if (proj.SubServices && proj.SubServices.length > 0) {
    // 将该项目的所有子服务加入 Tab 并激活首个子服务
    proj.SubServices.forEach(s => {
      const tabTitle = proj.SubServices.length > 1 ? `${proj.Name} · ${s.Name}` : proj.Name
      openTerminal(s.Id, tabTitle, s.Port, s.Status, false)
    })
    // 切换到第一个子服务
    activeTerminalId.value = proj.SubServices[0].Id
    isTerminalCollapsed.value = false
  }
}

const selectSubService = (proj, svc) => {
  selectedProjectId.value = proj.Id
  const tabTitle = proj.SubServices.length > 1 ? `${proj.Name} · ${svc.Name}` : proj.Name
  openTerminal(svc.Id, tabTitle, svc.Port, svc.Status, true)
}

const openTerminal = (serviceId, title, port, status, shouldActivate = true) => {
  isTerminalCollapsed.value = false
  if (shouldActivate) {
    activeTerminalId.value = serviceId
  }
  const exists = terminalTabs.value.find(t => t.id === serviceId)
  if (!exists) {
    terminalTabs.value.push({ id: serviceId, title, port, status })
  } else {
    // 更新标题确保为项目名
    exists.title = title
  }
}


const stopAllServices = async () => {
  if (confirm('确定停止所有正在运行的服务吗？')) {
    await fetch('/api/service/stop_all', { method: 'POST' })
  }
}

const isAppExited = ref(false)

const exitApplication = async () => {
  if (confirm('确定要退出 AppsManager 吗？\n这将同时安全停止所有正在运行的子服务。')) {
    try {
      await fetch('/api/system/exit', { method: 'POST' })
    } catch (e) {}
    isAppExited.value = true
    if (ws) {
      ws.close()
    }
  }
}

const closeWindow = () => {
  window.close()
  // 若浏览器不允许关闭标签，提示用户手动关闭
  setTimeout(() => {
    location.href = 'about:blank'
  }, 300)
}


const openServiceUrl = (port) => {
  if (port > 0) window.open(`http://localhost:${port}`, '_blank')
}

const openFolder = async (path) => {
  if (!path) return
  await fetch('/api/system/open_folder', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path })
  })
}

const openVSCode = async (path) => {
  if (!path) return
  await fetch('/api/system/open_code', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path })
  })
}

const openHBuilderX = async (path) => {
  if (!path) return
  await fetch('/api/system/open_hbuilderx', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path })
  })
}


const closeTerminalTab = (id) => {

  const idx = terminalTabs.value.findIndex(t => t.id === id)
  if (idx !== -1) {
    terminalTabs.value.splice(idx, 1)
    if (activeTerminalId.value === id) {
      activeTerminalId.value = terminalTabs.value[0]?.id || null
    }
  }
}

const openNewProjectModal = () => {
  isEditing.value = false
  form.value = {
    Id: crypto.randomUUID(),
    Name: '',
    Path: '',
    Group: '默认',
    SubServices: [
      { Id: crypto.randomUUID(), Name: '前端', StartCommand: 'pnpm dev', Port: 5173, Status: 0 }
    ]
  }
  showModal.value = true
}

const editProject = (proj) => {
  isEditing.value = true
  form.value = JSON.parse(JSON.stringify(proj))
  showModal.value = true
}

const addSubServiceRow = () => {
  form.value.SubServices.push({
    Id: crypto.randomUUID(),
    Name: '',
    StartCommand: '',
    Port: 0,
    Status: 0
  })
}

const removeSubServiceRow = (idx) => {
  form.value.SubServices.splice(idx, 1)
}

const selectFolder = async () => {
  try {
    const res = await fetch('/api/system/select_folder')
    const data = await res.json()
    if (data && data.path) {
      form.value.Path = data.path
      // 如果项目名称为空，自动提取文件夹名称作为项目名称
      if (!form.value.Name) {
        const parts = data.path.replace(/\\/g, '/').split('/').filter(Boolean)
        if (parts.length > 0) {
          form.value.Name = parts[parts.length - 1]
        }
      }
      // 自动触发智能识别
      await autoDetectServices()
    }
  } catch (e) {
    alert('选择文件夹失败: ' + e.message)
  }
}

const autoDetectServices = async () => {
  if (!form.value.Path) {
    alert('请先选择或输入物理路径')
    return
  }

  try {
    const res = await fetch('/api/projects/detect', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: form.value.Path })
    })
    const detected = await res.json()
    if (detected && detected.length > 0) {
      form.value.SubServices = detected
      const svcNames = detected.map(s => `• ${s.Name} (${s.StartCommand} :${s.Port})`).join('\n')
      alert(`🎉 穷举识别成功！共发现 ${detected.length} 个子启动项：\n\n${svcNames}`)
    } else {
      alert('未在此路径及多层子目录下检索到可自动启动的项目（已穷举扫描前端 Node/Vue/React、.NET Web/Worker、Python FastAPI/Flask/Django、Go、Java Spring Boot、Rust 等）。\n\n您可以点击【+ 添加启动项】快速手动配置。')
    }
  } catch (e) {
    alert('探测失败: ' + e.message)
  }
}

const saveProject = async () => {
  if (!form.value.Name || !form.value.Path) {
    alert('项目名称和路径不能为空')
    return
  }

  // 确保所有子服务都正确拥有物理路径
  form.value.SubServices.forEach(s => {
    if (!s.Path) {
      s.Path = form.value.Path
    }
  })

  let currentList = [...projects.value]
  if (isEditing.value) {
    const idx = currentList.findIndex(p => p.Id === form.value.Id)
    if (idx !== -1) currentList[idx] = form.value
  } else {
    currentList.push(form.value)
  }
  await fetch('/api/projects', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(currentList)
  })
  showModal.value = false
  fetchProjects()
}




const deleteProject = async (id) => {
  if (confirm('确定删除此项目配置吗？')) {
    const currentList = projects.value.filter(p => p.Id !== id)
    await fetch('/api/projects', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(currentList)
    })
    fetchProjects()
  }
}

const fetchSystemInfo = async () => {
  try {
    const res = await fetch('/api/system/info')
    const data = await res.json()
    if (data && data.localIp) {
      systemLocalIp.value = data.localIp
    }
  } catch (e) {}
}

onMounted(() => {
  fetchProjects()
  fetchSystemInfo()
  initWebSocket()
})
</script>


<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  background: var(--bg-main);
}
.app-header {
  height: 48px;
  background: #11141d;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 16px;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}
.logo-icon { font-size: 18px; }
.logo-title { font-size: 15px; font-weight: 700; color: #fff; letter-spacing: -0.3px; }
.version-tag { font-size: 11px; background: rgba(59, 130, 246, 0.15); color: #60a5fa; padding: 2px 6px; border-radius: 4px; }
.nav-tabs {
  display: flex;
  gap: 4px;
  margin-left: 16px;
  background: #11141c;
  padding: 3px;
  border-radius: 6px;
  border: 1px solid var(--border-color);
}
.nav-tab-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.nav-tab-btn:hover { color: #fff; }
.nav-tab-btn.active {
  background: #252b3b;
  color: #fff;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}
.stats-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1f2c;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  border: 1px solid #2d3348;
}
.header-right { display: flex; gap: 8px; }
.app-rules-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.app-main {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.app-sidebar {
  width: 200px;
  background: #0f121a;
  border-right: 1px solid var(--border-color);
  padding: 12px 8px;
  overflow-y: auto;
}
.sidebar-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding: 0 6px;
}
.sidebar-section-title {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 600;
  text-transform: uppercase;
}
.sidebar-add-btn {
  background: transparent;
  border: none;
  font-size: 11px;
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  transition: all 0.15s ease;
}
.sidebar-add-btn:hover {
  background: #1e2638;
  color: #60a5fa;
  transform: scale(1.1);
}
.group-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.15s ease;
  margin-bottom: 2px;
  position: relative;
  user-select: none;
}
.group-left-content {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}
.group-drag-handle {
  color: #94a3b8;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: grab;
  padding: 0 2px;
  opacity: 0.85;
  transition: all 0.15s ease;
  user-select: none;
}
.group-item:hover .group-drag-handle {
  opacity: 1;
  color: #60a5fa;
  transform: scale(1.15);
}
.group-item.active .group-drag-handle {
  color: #93c5fd;
  opacity: 1;
}

.group-name-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.group-right-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.group-hover-btns {
  display: none;
  align-items: center;
  gap: 2px;
}
.group-item:hover .group-hover-btns {
  display: flex;
}
.group-action-btn {
  background: transparent;
  border: none;
  font-size: 10px;
  padding: 1px 3px;
  border-radius: 3px;
  cursor: pointer;
  opacity: 0.7;
  transition: all 0.15s ease;
}
.group-action-btn:hover {
  opacity: 1 !important;
  background: #272f42;
  transform: scale(1.1);
}
.group-badge {
  font-size: 11px;
  color: var(--text-muted);
  min-width: 20px;
  text-align: right;
  font-family: monospace;
}
.group-item:hover { background: var(--bg-card); color: #fff; }
.group-item.active { background: #1e2638; color: #60a5fa; font-weight: 600; }

/* 拖拽放置与排序状态高亮 */
.group-item.drop-target-active {
  background: rgba(56, 189, 248, 0.2) !important;
  border: 1px dashed #38bdf8 !important;
  color: #fff !important;
  transform: scale(1.02);
  box-shadow: 0 0 10px rgba(56, 189, 248, 0.4);
}
.group-item.reorder-target-active {
  border-top: 2px solid #3b82f6 !important;
}
.group-item.group-dragging {
  opacity: 0.35;
}
.project-card.card-is-dragging {
  opacity: 0.4;
  transform: scale(0.98);
  border: 1px dashed #38bdf8;
}


.app-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.cards-container {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
  align-content: start;
}
.project-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: all 0.2s ease;
}
.project-card:hover { border-color: #3b4256; background: var(--bg-card-hover); }
.project-card.card-active { border-color: #3b82f6; box-shadow: 0 0 12px rgba(59, 130, 246, 0.2); }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.card-title-group { display: flex; align-items: center; gap: 8px; }
.project-name { font-size: 14px; font-weight: 600; color: #fff; }
.copy-svg-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #181e2b;
  border: 1px solid #2d3548;
  color: #94a3b8;
  padding: 3px 4px;
  border-radius: 4px;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.15s ease;
  line-height: 1;
}
.copy-svg-btn:hover {
  background: #2563eb;
  border-color: #3b82f6;
  color: #fff;
  transform: scale(1.1);
  box-shadow: 0 0 6px rgba(59, 130, 246, 0.4);
}
.badge { font-size: 11px; background: #23293a; color: var(--text-secondary); padding: 2px 6px; border-radius: 4px; }
.card-actions { display: flex; gap: 4px; }
.icon-btn {
  background: transparent;
  border: none;
  font-size: 13px;
  padding: 4px;
  border-radius: 4px;
  cursor: pointer;
  opacity: 0.7;
}
.icon-btn:hover { opacity: 1; background: #272f42; }
.btn-danger-hover:hover { background: rgba(239, 68, 68, 0.2); }
.card-path-row { display: flex; align-items: center; gap: 6px; width: 100%; }
.card-path {
  font-size: 11px;
  color: var(--text-muted);
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: calc(100% - 28px);
  flex-shrink: 1;
}

/* 全局 Toast 提示 */
.global-toast {
  position: fixed;
  top: 60px;
  left: 50%;
  transform: translateX(-50%);
  background: #1e293b;
  color: #38bdf8;
  border: 1px solid #38bdf8;
  padding: 8px 18px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  z-index: 2000;
  pointer-events: none;
}
.toast-enter-active, .toast-leave-active { transition: all 0.25s ease; }
.toast-enter-from, .toast-leave-to { opacity: 0; transform: translate(-50%, -10px); }



.sub-services-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: #0f121a;
  padding: 8px;
  border-radius: 6px;
}
.svc-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 5px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
}
.svc-row:hover { background: #181e2b; }
.svc-row-active { background: #1c2436; border: 1px solid rgba(59, 130, 246, 0.4); }
.svc-info { display: flex; align-items: center; gap: 8px; font-size: 12px; }
.dot { width: 8px; height: 8px; border-radius: 50%; }

.dot-running { background: #10b981; }
.dot-starting { background: #f59e0b; }
.dot-stopped { background: #4b5563; }
.svc-port-badge {
  font-family: monospace;
  font-size: 11px;
  background: #1c2333;
  color: #94a3b8;
  padding: 1px 6px;
  border-radius: 4px;
  cursor: pointer;
}
.svc-port-badge.port-active { color: #10b981; background: rgba(16, 185, 129, 0.15); }
.svc-controls { display: flex; gap: 4px; }
.action-btn {
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 4px;
  border: 1px solid var(--border-color);
  background: #1c2130;
  color: var(--text-secondary);
  cursor: pointer;
}
.action-btn:hover { background: #272f42; color: #fff; }
.btn-run { background: rgba(16, 185, 129, 0.15); color: #34d399; border-color: rgba(16, 185, 129, 0.3); }
.btn-run:hover { background: #10b981; color: #fff; }
.btn-stop { background: rgba(239, 68, 68, 0.15); color: #f87171; border-color: rgba(239, 68, 68, 0.3); }
.btn-stop:hover { background: #ef4444; color: #fff; }
.btn-bolt { color: #f59e0b; }
.btn-bolt:hover { background: rgba(245, 158, 11, 0.2); }
.btn-active { background: #3b82f6 !important; color: #fff !important; }

/* 底部抽屉终端样式 */
.terminal-drawer {
  position: relative;
  background: #0d1017;
  border-top: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
}
.terminal-drawer.collapsed { height: 36px !important; }
.drawer-resizer {
  position: absolute;
  top: -4px;
  left: 0;
  right: 0;
  height: 8px;
  cursor: row-resize;
  z-index: 50;
  transition: background 0.2s ease;
}
.drawer-resizer:hover, .drawer-resizer:active {
  background: #3b82f6;
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.6);
}
.drawer-header {
  height: 36px;
  background: #141721;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 12px;
}

.drawer-tabs { display: flex; gap: 4px; align-items: center; overflow-x: auto; }
.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: #1c2130;
  border-radius: 4px;
  font-size: 11px;
  cursor: pointer;
  color: var(--text-secondary);
}
.tab-item.active { background: #2d3748; color: #fff; font-weight: 500; }
.tab-close { font-size: 14px; opacity: 0.6; margin-left: 4px; }
.tab-close:hover { opacity: 1; color: #f87171; }
.empty-tabs { font-size: 11px; color: var(--text-muted); }
.drawer-body { flex: 1; overflow: hidden; }
.terminal-pane { height: 100%; width: 100%; }

/* 按钮通用 */
.btn {
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid transparent;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn-primary { background: #2563eb; color: #fff; }
.btn-primary:hover { background: #1d4ed8; }
.btn-secondary { background: #1e2433; color: #e2e8f0; border-color: #2d3548; }
.btn-secondary:hover { background: #2a3348; }
.btn-warning { background: rgba(245, 158, 11, 0.15); color: #fbbf24; border-color: rgba(245, 158, 11, 0.3); }
.btn-warning:hover:not(:disabled) { background: #d97706; color: #fff; }
.btn-warning:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-danger { background: rgba(239, 68, 68, 0.15); color: #f87171; border-color: rgba(239, 68, 68, 0.3); }
.btn-danger:hover:not(:disabled) { background: #ef4444; color: #fff; }
.btn-danger:disabled { opacity: 0.4; cursor: not-allowed; }
.btn-sm { padding: 3px 8px; font-size: 11px; border-radius: 4px; cursor: pointer; }


/* 弹窗 Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-card {
  width: 580px;
  background: #141721;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
}
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 14px 16px; border-bottom: 1px solid var(--border-color); }
.modal-body { padding: 16px; display: flex; flex-direction: column; gap: 12px; max-height: 70vh; overflow-y: auto; }
.modal-footer { padding: 12px 16px; border-top: 1px solid var(--border-color); display: flex; justify-content: flex-end; gap: 8px; }
.form-group { display: flex; flex-direction: column; gap: 6px; }
.form-group label { font-size: 12px; color: var(--text-secondary); font-weight: 500; }
.form-input {
  background: #0d1017;
  border: 1px solid #2d3348;
  border-radius: 4px;
  padding: 6px 10px;
  color: #fff;
  font-size: 12px;
  outline: none;
}
.form-input:focus { border-color: #3b82f6; }
.input-with-btn { display: flex; gap: 8px; }
.input-with-btn .form-input { flex: 1; }
.group-select-row { display: flex; gap: 8px; width: 100%; }
.form-select {
  width: 100%;
  cursor: pointer;
  appearance: auto;
}
.form-select option {
  background: #141721;
  color: #fff;
  padding: 6px 10px;
}
.sub-header { display: flex; justify-content: space-between; align-items: center; }
.sub-table { display: flex; flex-direction: column; gap: 6px; }
.sub-edit-row { display: flex; gap: 6px; align-items: center; }
.flex-1 { flex: 1; }
.flex-2 { flex: 2; }
.flex-0-8 { flex: 0.8; }
</style>








