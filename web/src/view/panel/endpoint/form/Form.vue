<template>
    <div class="endpoint">
        <div class="form-row">
            <span class="form-label">启用</span>
            <Toggle v-model="form.Enable" />
        </div>
        <div class="form-row half">
            <span class="form-label">标签</span>
            <Input v-model="form.Tag" placeholder="tag" :disabled="form.Tag === 'warp'" />
        </div>
        <div class="form-row half">
            <span class="form-label">协议</span>
            <Select v-model="form.Type" :options="protocols.map(value => ({label: value, value}))" />
        </div>
        <div class="form-row"><span class="form-label">上游中继</span><Input v-model="form.Detour" placeholder="中继节点标签，空为直连拨号" /></div>
        <template v-if="form.Type !== 'wireguard'">
            <div class="form-row"><span class="form-label">地址</span><Input v-model="form.Server" /></div>
            <div class="form-row"><span class="form-label">端口</span><Input v-model="form.ServerPort" type="number" :min="1" :max="65535" /></div>
            <div v-if="['socks','http','ssh'].includes(form.Type)" class="form-row"><span class="form-label">{{ form.Type === 'ssh' ? 'SSH 用户' : '用户名' }}</span><Input v-model="form.Username" /></div>
            <div v-if="!['vless','vmess'].includes(form.Type)" class="form-row"><span class="form-label">密码</span><Input v-model="form.Password" type="password" /></div>
            <div v-for="field in fields[form.Type] || []" :key="field" class="form-row">
                <span class="form-label">{{ field }}</span><Input :model-value="options[field] ?? ''" @update:model-value="setOption(field, $event)" />
            </div>
            <p v-if="form.Type === 'ssh'">Host Key 应与协商算法匹配，例如 host_key_algorithms 使用 ssh-ed25519。留空 Host Key 时不校验服务器身份。</p>
            <div class="form-row" v-if="form.Type === 'ssh'"><span class="form-label">私钥 PEM</span><textarea :value="options.private_key || ''" @input="setOption('private_key', ($event.target as HTMLTextAreaElement).value)" /></div>
            <p>参数遵循 sing-box 1.13.12。HTTP/SSH 仅 TCP；SOCKS4/4a 不支持 UDP。TLS、Reality、传输层及 UDP 参数可在下方 JSON 中编辑，保存时由核心检查。</p>
            <label>协议 Options JSON（含凭据，请妥善保管）<textarea v-model="form.Options" spellcheck="false" rows="10" /></label>
            <button class="action-btn" type="button" @click="setDefaults">填入当前协议示例</button>
        </template>
        <template v-if="['socks','http'].includes(form.Type)">
            <label>动态策略 <Select v-model="dynamicMode" :options="['fixed','session','api'].map(value => ({label:value,value}))" /></label>
            <p>会话模板使用 {session}、{username}、{password}；ttl_seconds 为有效秒数。API mapping 将 JSON 字段路径映射至 server/server_port/username/password。rotate_on_failure 仅对连续传输故障轮换，目标站 403/429 不触发。</p>
            <textarea v-model="form.Dynamic" spellcheck="false" rows="8" aria-label="动态代理策略 JSON" />
        </template>
        <template v-if="form.Type === 'wireguard'">
            <label>WireGuard Options（system/name/listen_port/pre_shared_key/persistent_keepalive_interval）<textarea v-model="form.Options" spellcheck="false" rows="4" /></label>
            <div class="form-row">
                <span class="form-label">本地 IPv4 地址</span>
                <Input v-model="form.Address[0]" placeholder="0.0.0.0/32" />
            </div>
            <div class="form-row">
                <span class="form-label">本地 IPv6 地址</span>
                <Input v-model="form.Address[1]" placeholder="::/128" />
            </div>
            <div class="form-row">
                <span class="form-label">公钥</span>
                <Input v-model="form.PublicKey" readonly />
            </div>
            <div class="form-row">
                <span class="form-label">私钥</span>
                <Input v-model="form.PrivateKey" readonly />
            </div>
            <div class="form-row">
                <span class="form-label"></span>
                <button class="action-btn" @click="genKeyPair">生成密钥对</button>
            </div>
            <div class="form-row quarter">
                <span class="form-label">MTU</span>
                <Input v-model="form.MTU" type="number" :min="0" placeholder="1408" />
            </div>
            <div class="form-row">
                <span class="form-label">对端地址</span>
                <Input
                    :model-value="form.PeerAddress + ':' + form.PeerPort"
                    @update:model-value="
                        (val) => {
                            const parts = String(val).split(':')
                            form.PeerAddress = parts[0]
                            form.PeerPort = parseInt(parts[1] ?? '0') || 0
                        }
                    "
                    placeholder="engage.cloudflareclient.com:2408"
                />
            </div>
            <div class="form-row">
                <span class="form-label">Reserved</span>
                <Input v-model="reserved" placeholder="0,0,0" />
            </div>
            <div class="form-row">
                <span class="form-label">对端公钥</span>
                <Input v-model="form.PeerPublicKey" />
            </div>
            <div class="form-row">
                <span class="form-label">允许的IP</span>
                <Input v-model="form.AllowedIPs" placeholder="0.0.0.0/0,::/0" disabled />
            </div>
            <div class="form-row quarter">
                <span class="form-label">UDP 超时 (s)</span>
                <Input
                    v-model="form.UDPTimeout"
                    type="number"
                    :min="0"
                    placeholder="秒，0 表示默认"
                />
            </div>
            <div class="form-row quarter">
                <span class="form-label">Workers</span>
                <Input v-model="form.Workers" type="number" :min="0" placeholder="0 表示默认" />
            </div>
        </template>
    </div>
</template>

<script setup lang="ts">
import Toggle from '@/component/ui/Toggle.vue'
import Input from '@/component/ui/Input.vue'
import Select from '@/component/ui/Select.vue'
import { generateWireguardKeyPair } from '@/api/generate'

const form = defineModel<any>({
    default: () => ({
        Enable: true,
        Tag: '',
        Type: 'wireguard',
        Address: ['', ''],
        PublicKey: '',
        PrivateKey: '',
        MTU: 1408,
        Reserved: '',
        PeerAddress: '',
        PeerPort: 0,
        PeerPublicKey: '',
        AllowedIPs: '0.0.0.0/0,::/0',
        UDPTimeout: 0,
        Workers: 0,
    }),
})

const protocols = ['socks','http','ssh','shadowsocks','vless','vmess','trojan','hysteria2','tuic','anytls','wireguard','custom']
const fields: Record<string, string[]> = {
    socks: ['version'], ssh: ['private_key_passphrase','host_key','host_key_algorithms'], shadowsocks: ['method'],
    vless: ['uuid','flow'], vmess: ['uuid','security'], tuic: ['uuid','congestion_control','udp_relay_mode'],
}
const options = computed(() => { try { return JSON.parse(form.value.Options || '{}') } catch { return {} } })
function setOption(field: string, value: unknown) {
    form.value.Options = JSON.stringify({...options.value, [field]: value}, null, 2)
}
function setDefaults() {
    const defaults: Record<string, any> = {
        socks: {version:'5'}, http: {tls:{enabled:true,server_name:form.value.Server}}, ssh: {host_key:[]},
        shadowsocks: {method:'aes-128-gcm'}, vless: {uuid:'',tls:{enabled:true}}, vmess:{uuid:'',security:'auto'},
        trojan:{tls:{enabled:true}}, hysteria2:{tls:{enabled:true}, up_mbps:100,down_mbps:100},
        tuic:{uuid:'',tls:{enabled:true,alpn:['h3']},congestion_control:'cubic',udp_relay_mode:'native'},
        anytls:{tls:{enabled:true},idle_session_timeout:'30s'}, custom:{type:'socks',version:'5'},
    }
    form.value.Options=JSON.stringify(defaults[form.value.Type] || {},null,2)
}

const dynamicMode = computed({
    get: () => {try {return JSON.parse(form.value.Dynamic || '{}').mode || 'fixed'} catch {return 'fixed'}},
    set: (mode: string) => {form.value.Dynamic=JSON.stringify(mode==='session' ? {mode,session_id:'',ttl_seconds:600,username_template:'{username}-session-{session}',rotate_on_failure:false} : mode==='api' ? {mode,url:'',method:'POST',headers:{},body:'{}',mapping:{server:'data.host',server_port:'data.port'},rotate_on_failure:false} : {mode}, null, 2)},
})
const reserved = computed({
    get() {
        if (!form.value.Reserved) return ''
        try {
            return JSON.parse(form.value.Reserved).join(',')
        } catch {
            return form.value.Reserved
        }
    },
    set(val: string) {
        const nums = val
            .split(',')
            .map((s) => parseInt(s.trim()))
            .filter((n) => !isNaN(n))
        form.value.Reserved = JSON.stringify(nums)
    },
})

async function genKeyPair() {
    const res = await generateWireguardKeyPair()
    form.value.PrivateKey = res.private_key
    form.value.PublicKey = res.public_key
}
</script>

<style scoped>
textarea { width: 100%; min-height: 90px; background: var(--color-bg); color: var(--color-text-light); padding: 12px; border-radius: 10px; }
.endpoint {
    display: flex;
    flex-direction: column;
    gap: 20px;
    padding: 20px;
}

:deep(.section) {
    border-radius: 0;

    .header {
        background-color: var(--color-bg);
    }
    .body {
        .content {
            display: flex;
            flex-direction: column;
            gap: 10px;
            padding: 20px;
        }
    }
}

.form-row {
    display: flex;
    align-items: center;
    gap: 16px;

    button {
        font-size: var(--font-size-sm);
    }

    &.half :deep(.input-wrap),
    &.half :deep(.select) {
        max-width: 50%;
    }

    &.quarter :deep(.input-wrap),
    &.quarter :deep(.select) {
        max-width: 25%;
    }
}

.form-label {
    width: 100px;
    flex-shrink: 0;
    font-size: var(--font-size-sm);
    color: var(--color-text-dark);
    text-align: right;
}

.action-btn {
    gap: 5px;
    padding: 8px 16px;
}

.pair {
    display: flex;
    gap: 2px;
    flex: 1;

    &:deep(.input-wrap) {
        &:first-child .input {
            border-radius: 999px 0 0 999px;
        }
        &:last-child .input {
            border-radius: 0 999px 999px 0;
        }
    }
}
</style>
