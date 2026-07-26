import { createRouter, createWebHistory } from 'vue-router'
import OperationsView from './views/OperationsView.vue'
import DevicesView from './views/DevicesView.vue'
import AlarmsView from './views/AlarmsView.vue'
import CommandsView from './views/CommandsView.vue'
import PlaceholderView from './views/PlaceholderView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/app/demo/overview' },
    { path: '/app/:workspaceId/overview', component: OperationsView },
    { path: '/app/:workspaceId/devices', component: DevicesView },
    { path: '/app/:workspaceId/alarms', component: AlarmsView },
    { path: '/app/:workspaceId/commands', component: CommandsView },
    { path: '/app/:workspaceId/replay/:deviceId?', component: PlaceholderView, props: { title: '轨迹回放', eyebrow: 'Replay' } },
    { path: '/app/:workspaceId/operations', component: PlaceholderView, props: { title: '系统运行', eyebrow: 'Operations' } },
    { path: '/:pathMatch(.*)*', redirect: '/app/demo/overview' },
  ],
})

export default router
