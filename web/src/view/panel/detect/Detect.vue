<template>
    <main>
        <div class="page">
            <div class="body">
                <div class="detect"><label>检测出口 <Select v-model="nodeID" :options="nodeOptions" /></label><p>IP 与解锁请求均经过所选节点。下方 MTR 始终检测服务器回程。</p></div>
                <IPDetect :key="`ip-${nodeID}`" :node-id="Number(nodeID)" />
                <UnlockDetect :key="`unlock-${nodeID}`" :node-id="Number(nodeID)" />
                <p>服务器级 MTR 回程（SOCKS / SSH 等节点不支持此项）</p><BackRouteDetect />
                <details class="detect"><summary>所选节点历史</summary><button @click="loadHistory">刷新历史</button><p v-for="e in events" :key="`event-${e.ID}`">{{ e.CreatedAt }} · {{ e.Kind }} · {{ e.BeforeIP }} → {{ e.AfterIP }} · {{ e.Status }}</p><p v-for="r in history" :key="r.ID">{{ r.UpdatedAt }} · {{ r.NodeTag || '服务器直连' }} · {{ r.IP }} · {{ r.ASN }} · {{ r.Country }} {{ r.City }} · {{ r.Source }}</p></details>
            </div>
        </div>
    </main>
</template>

<script setup lang="ts">
import Select from '@/component/ui/Select.vue'
import { request } from '@/util/request'
import { getEndpoints } from '@/api/endpoint'
import { getIP } from '@/api/detect'
import IPDetect from '@/view/panel/detect/widget/IP.vue'
import UnlockDetect from '@/view/panel/detect/widget/Unlock.vue'
import BackRouteDetect from '@/view/panel/detect/widget/BackRoute.vue'
const nodeID = ref(0)
const nodes = ref<any[]>([])
const history = ref<any[]>([])
const events = ref<any[]>([])
const nodeOptions = computed(() => [{label:'服务器直连',value:0},...nodes.value.filter(n => n.Enable).map(n => ({label:n.Tag,value:n.ID}))])
async function loadHistory(){ const selected=Number(nodeID.value);const [ips,ev]=await Promise.all([getIP(selected),request(`/api/detect/history?node_id=${selected}`)]);if(selected!==Number(nodeID.value))return;history.value=ips.data||[];events.value=ev.data||[] }
onMounted(async()=>{nodes.value=(await getEndpoints()).data;await loadHistory()})
watch(nodeID,loadHistory)
</script>

<style scoped>
:deep(.detect) {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
    padding: 20px;
    background-color: var(--color-bg-dark);
    border-radius: 20px;
    box-shadow: var(--box-shadow);

    .header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-wrap: wrap;
        gap: 10px;

        .title-wrap {
            display: flex;
            align-items: flex-end;
            gap: 5px;

            .title {
                font-weight: bold;
                color: var(--color-text-light);
            }

            .updated {
                font-size: var(--font-size-sm);
                color: var(--color-text-dark);
            }
        }

        .action {
            display: flex;
            align-items: center;
            gap: 5px;
            margin-left: auto;

            > button {
                gap: 5px;
                padding: 5px 15px 5px 10px;
                font-size: var(--font-size-sm);
                background-color: var(--color-bg-light);
                color: var(--color-text-light);

                &:hover:not(:disabled) {
                    background-color: var(--color-bg);
                }

                .icon {
                    font-size: 12px;
                }

                @media (max-width: 768px) {
                    padding: 6px;

                    span {
                        display: none;
                    }
                }
            }
        }
    }
}
</style>
