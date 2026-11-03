<!-- Endpoint.vue -->
<template>
    <main>
        <div class="page">
            <div class="body">
                <button class="add-btn" @click="create">
                    <i class="icon">add</i>
                    添加节点
                </button>
                <button class="add-btn" @click="showWarp = true">WARP</button>

                <button class="add-btn" @click="showTransfer = true">导入 / 导出出口配置</button>
                <List :headers="['标签', '协议', '地址', '启用', '操作']">
                    <tr v-for="ep in endpoints" :key="ep.ID">
                        <td>{{ ep.Tag }}</td>
                        <td>
                            <span
                                class="tag"
                                :class="ep.Type === 'wireguard' ? 'primary' : 'green'"
                            >
                                {{ ep.Type }}
                            </span>
                        </td>
                        <td>{{ ep.Type === 'wireguard' ? ep.PeerAddress : ep.Server }}:{{ ep.Type === 'wireguard' ? ep.PeerPort : ep.ServerPort }}</td>
                        <td>
                            <Toggle :model-value="ep.Enable" @update:model-value="toggle(ep)" />
                        </td>
                        <td class="actions">
                            <button class="icon-btn" @click="openRoute(ep)" title="路由">
                                <i class="icon">alt_route</i>
                            </button>
                            <button class="icon-btn" @click="rotate(ep)" title="轮换 IP"><i class="icon">refresh</i></button>
                            <button class="icon-btn" @click="edit(ep)" title="编辑">
                                <i class="icon">edit</i>
                            </button>
                            <button class="icon-btn danger" @click="remove(ep.ID)" title="删除">
                                <i class="icon">delete</i>
                            </button>
                        </td>
                    </tr>
                    <tr v-if="endpoints.length === 0">
                        <td colspan="5" class="empty">暂无节点</td>
                    </tr>
                </List>
            </div>
        </div>
    </main>
    <Drawer v-model="showTransfer" title="受管理的代理出口配置（包含敏感凭据）" :footer="false">
        <div class="transfer">
            <p>此处导出服务器持有的出口认证。用户订阅仍只描述 SLINX-EF 入口。sing-box / SLINX-EF 包含完整中继依赖；URI 不支持中继。导入支持 SLINX-EF、sing-box 节点 JSON、常见 URI，重复标签会拒绝。</p>
            <Select v-model="exportFormat" :options="['slinx-ef','sing-box','mihomo','surge','uri'].map(value => ({label:value,value}))" />
            <label v-for="ep in endpoints" :key="ep.ID"><input v-model="exportTags" type="checkbox" :value="ep.Tag" /> {{ ep.Tag }}</label>
            <button @click="exportConfig">导出所选节点</button>
            <textarea v-model="transferContent" rows="18" spellcheck="false" aria-label="节点导入导出内容" />
            <button @click="downloadConfig">下载配置</button><button @click="importConfig">导入文本中的节点并应用</button>
            <p>{{ transferStatus }}</p>
        </div>
    </Drawer>
    <Drawer v-model="showWarp" title="Cloudflare WARP" :footer="false">
        <Warp v-if="showWarp" @applied="load" />
    </Drawer>
    <Drawer v-model="showDrawer" :title="drawerTitle" @save="save">
        <Form v-model="defaultEndpoint" />
    </Drawer>

    <Drawer v-model="showRoute" title="路由设置" @save="routeRef?.save()">
        <Route v-if="showRoute" ref="routeRef" :endpoint="routeEndpoint" />
    </Drawer>
</template>

<script setup lang="ts">
import { request } from '@/util/request'
import Select from '@/component/ui/Select.vue'
import Drawer from '@/component/Drawer.vue'
import List from '@/component/ui/List.vue'
import Toggle from '@/component/ui/Toggle.vue'
import Warp from '@/view/panel/endpoint/widget/Warp.vue'
import Form from '@/view/panel/endpoint/form/Form.vue'
import { getEndpoints, saveEndpoint, deleteEndpoint, toggleEndpoint } from '@/api/endpoint'
import { generateWireguardKeyPair } from '@/api/generate'

const showTransfer=ref(false)
const exportFormat=ref('slinx-ef')
const exportTags=ref<string[]>([])
const transferContent=ref('')
const transferStatus=ref('')
async function exportConfig(){try{transferContent.value=(await request('/api/endpoint/export',{method:'POST',body:JSON.stringify({tags:exportTags.value,format:exportFormat.value})})).data;transferStatus.value='已导出，包含敏感凭据'}catch(err:any){transferStatus.value=err.error}}
async function importConfig(){try{const res=await request('/api/endpoint/import',{method:'POST',body:JSON.stringify({content:transferContent.value})});transferStatus.value=`已导入并应用 ${res.count} 个节点`;await load()}catch(err:any){transferStatus.value=err.error}}
function downloadConfig(){const url=URL.createObjectURL(new Blob([transferContent.value],{type:'text/plain'}));const a=document.createElement('a');a.href=url;a.download=`slinx-ef-nodes.${exportFormat.value==='mihomo'?'yaml':exportFormat.value==='surge'?'conf':exportFormat.value==='uri'?'txt':'json'}`;a.click();URL.revokeObjectURL(url)}
const modal = inject<any>('modal')

// ── 状态 ──────────────────────────────────────────────────

const endpoints = ref<any[]>([])
const showWarp = ref(false)
const showDrawer = ref(false)
const defaultEndpoint = ref<any>(baseEndpoint())
const drawerTitle = ref('添加节点')

// ── 工具函数 ───────────────────────────────────────────────

function baseEndpoint() {
    return {
        Enable: true,
        Tag: '',
        Type: 'socks',
        Options: '{}', Detour: '', Dynamic: '',
 Server: '', ServerPort: 1080, Username: '', Password: '',
        MTU: 1408,
        Address: ['172.16.0.2/32', 'fd00::2/128'],
        PrivateKey: '',
        PublicKey: '',
        PeerAddress: '',
        PeerPort: 2408,
        PeerPublicKey: '',
        AllowedIPs: '0.0.0.0/0,::/0',
        Reserved: '[0,0,0]',
        UDPTimeout: 0,
        Workers: 0,
    }
}

// ── 生命周期 ───────────────────────────────────────────────

onMounted(() => {
    load()
})

// ── 列表操作 ───────────────────────────────────────────────

async function load() {
    const res = await getEndpoints()
    endpoints.value = res.data
}

async function rotate(ep: any) {
    try {const res=await request(`/api/endpoint/${ep.ID}/rotate`,{method:'POST'});modal.value?.show('success',res.data.Status);await load()}
    catch(err:any){modal.value?.show('error',err.error)}
}
async function toggle(ep: any) {
    await toggleEndpoint(ep.ID)
    await load()
}

function remove(id: number) {
    modal.value?.show('confirm', '确认删除该节点？', async () => {
        await deleteEndpoint(id)
        await load()
    })
}

// ── Drawer 操作 ────────────────────────────────────────────

async function create() {
    drawerTitle.value = '添加节点'
    defaultEndpoint.value = baseEndpoint()
    showDrawer.value = true

    const res = await generateWireguardKeyPair()
    defaultEndpoint.value = {
        ...defaultEndpoint.value,
        PrivateKey: res.private_key,
        PublicKey: res.public_key,
    }
}

async function edit(ep: any) {
    ep = (await request(`/api/endpoint/${ep.ID}`)).data
    defaultEndpoint.value = {
        ...defaultEndpoint.value,
        ...ep,
        Address: JSON.parse(ep.Address || '[]'),
    }
    drawerTitle.value = '编辑节点'
    showDrawer.value = true
}

async function save() {
    try {
        const payload = {
            ...defaultEndpoint.value,
            Address: JSON.stringify(defaultEndpoint.value.Address),
        }
        const res = await saveEndpoint(payload)
        modal.value?.show('success', `配置已保存；核心状态：${res.core_status || 'unknown'}`)
        await load()
        showDrawer.value = false
    } catch (err: any) {
        modal.value?.show('error', err?.error)
    }
}

import Route from '@/view/panel/endpoint/widget/Route.vue'

const showRoute = ref(false)
const routeEndpoint = ref<any>(null)
const routeRef = ref<any>(null)

function openRoute(ep: any) {
    routeEndpoint.value = ep
    showRoute.value = true
}
</script>


<style scoped>
.transfer {display:flex;flex-direction:column;gap:12px;padding:20px}
.transfer textarea {width:100%;background:var(--color-bg);color:var(--color-text-light);padding:12px;border-radius:10px}
</style>
