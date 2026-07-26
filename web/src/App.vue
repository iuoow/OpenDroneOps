<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import ConnectionIndicator from './components/ConnectionIndicator.vue'
import StatusBadge from './components/StatusBadge.vue'
import { useOperationsStore } from './state/operations'

const route = useRoute()
const store = useOperationsStore()
const workspaceId = computed(() => String(route.params.workspaceId ?? store.workspaceId))

const navigation = [
  { label: '实时态势', short: 'OV', path: 'overview' },
  { label: '设备管理', short: 'DV', path: 'devices' },
  { label: '告警中心', short: 'AL', path: 'alarms' },
  { label: '指令中心', short: 'CM', path: 'commands' },
  { label: '轨迹回放', short: 'RP', path: 'replay' },
  { label: '系统运行', short: 'OP', path: 'operations' },
]

onMounted(() => store.hydrate(workspaceId.value))
onUnmounted(() => store.stopRealtime())
</script>

<template>
  <div class="app-shell">
    <header class="topbar">
      <RouterLink class="brand" to="/app/demo/overview" aria-label="OpenDroneOps 实时态势首页">
        <span class="brand__mark" aria-hidden="true">OD</span>
        <span>
          <strong>OpenDroneOps</strong>
          <small>OPERATIONS</small>
        </span>
      </RouterLink>
      <div class="workspace-switcher" aria-label="当前 Workspace">
        <span class="eyebrow">WORKSPACE</span>
        <strong>{{ workspaceId }}</strong>
        <span class="workspace-switcher__chevron" aria-hidden="true">⌄</span>
      </div>
      <label class="global-search">
        <span class="sr-only">搜索设备、告警或指令</span>
        <span aria-hidden="true">⌕</span>
        <input placeholder="搜索设备、告警或指令" />
        <kbd>⌘ K</kbd>
      </label>
      <div class="topbar__actions">
        <ConnectionIndicator :status="store.connection" :detail="store.connectionDetail" />
        <RouterLink class="critical-counter" to="/app/demo/alarms" aria-label="查看严重告警">
          <span aria-hidden="true">!</span>
          {{ store.criticalCount }} 严重
        </RouterLink>
        <button class="avatar-button" type="button" aria-label="打开用户菜单">OD</button>
      </div>
    </header>

    <aside class="sidebar" aria-label="主导航">
      <nav>
        <RouterLink
          v-for="item in navigation"
          :key="item.path"
          class="nav-item"
          :to="`/app/${workspaceId}/${item.path}`"
          :aria-label="item.label"
        >
          <span class="nav-item__icon" aria-hidden="true">{{ item.short }}</span>
          <span class="nav-item__label">{{ item.label }}</span>
        </RouterLink>
      </nav>
      <div class="sidebar__footer">
        <span class="sidebar__status-dot" aria-hidden="true"></span>
        <span>Demo 数据源</span>
      </div>
    </aside>

    <main class="main-content">
      <div v-if="store.error" class="global-notice" role="alert">
        <strong>快照加载失败</strong>
        <span>{{ store.error }}</span>
        <button type="button" @click="store.hydrate(workspaceId)">重试</button>
      </div>
      <RouterView v-slot="{ Component }">
        <component :is="Component" />
      </RouterView>
    </main>

    <div class="sr-only" aria-live="polite">
      {{ store.connection === 'recovering' ? '正在恢复实时数据' : '' }}
    </div>
  </div>
</template>
