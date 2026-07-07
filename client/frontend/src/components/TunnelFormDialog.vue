<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { type AddTunnelInput } from '@/api'
import { useServerStore } from '@/stores/server'

const props = defineProps<{
  visible: boolean
  serverId: string
}>()
const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'submit', input: AddTunnelInput): void
}>()

const serverStore = useServerStore()
const formRef = ref<FormInstance>()

const form = reactive<{
  name: string
  protocol: 'tcp' | 'udp' | 'http' | 'https'
  local_ip: string
  local_port: number
  remote_port: number
  auto_port: boolean
  domain_mode: 'custom' | 'subdomain'
  custom_domain: string
  subdomain: string
}>({
  name: '', protocol: 'tcp', local_ip: '127.0.0.1', local_port: 0,
  remote_port: 0, auto_port: false,
  domain_mode: 'custom', custom_domain: '', subdomain: '',
})

const isPortProtocol = computed(() => form.protocol === 'tcp' || form.protocol === 'udp')
const isDomainProtocol = computed(() => form.protocol === 'http' || form.protocol === 'https')

const rules: FormRules = {
  name: [{ required: true, message: '请输入映射名称', trigger: 'blur' }],
  local_port: [{ required: true, message: '请输入本地端口', trigger: 'blur' }],
}

watch(() => props.visible, (v) => {
  if (v) {
    Object.assign(form, {
      name: '', protocol: 'tcp', local_ip: '127.0.0.1', local_port: 0,
      remote_port: 0, auto_port: false,
      domain_mode: 'custom', custom_domain: '', subdomain: '',
    })
  }
})

function close() {
  emit('update:visible', false)
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    const input: AddTunnelInput = {
      server_id: props.serverId,
      name: form.name,
      protocol: form.protocol,
      local_ip: form.local_ip,
      local_port: form.local_port,
    }
    if (isPortProtocol.value) {
      input.remote_port = form.remote_port
    } else {
      if (form.domain_mode === 'custom') {
        input.custom_domain = form.custom_domain
      } else {
        input.subdomain = form.subdomain
      }
    }
    emit('submit', input)
  })
}
</script>

<template>
  <el-dialog :model-value="visible" title="创建映射" width="520px" @update:model-value="close">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
      <el-form-item label="映射名称" prop="name"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="协议">
        <el-radio-group v-model="form.protocol">
          <el-radio-button label="tcp">TCP</el-radio-button>
          <el-radio-button label="udp">UDP</el-radio-button>
          <el-radio-button label="http">HTTP</el-radio-button>
          <el-radio-button label="https">HTTPS</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="本地 IP"><el-input v-model="form.local_ip" /></el-form-item>
      <el-form-item label="本地端口" prop="local_port"><el-input-number v-model="form.local_port" :min="1" :max="65535" /></el-form-item>

      <template v-if="isPortProtocol">
        <el-form-item label="自动分配端口"><el-switch v-model="form.auto_port" /></el-form-item>
        <el-form-item label="远程端口" v-if="!form.auto_port">
          <el-input-number v-model="form.remote_port" :min="1" :max="65535" />
        </el-form-item>
      </template>

      <template v-if="isDomainProtocol">
        <el-form-item label="域名模式">
          <el-radio-group v-model="form.domain_mode">
            <el-radio label="custom">自定义域名</el-radio>
            <el-radio label="subdomain">子域名前缀</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="自定义域名" v-if="form.domain_mode === 'custom'">
          <el-input v-model="form.custom_domain" placeholder="app.example.com" />
        </el-form-item>
        <el-form-item label="子域名前缀" v-if="form.domain_mode === 'subdomain'">
          <el-input v-model="form.subdomain" placeholder="demo" />
          <div class="hint" v-if="serverStore.capabilities?.subdomain_host">
            最终域名：{{ form.subdomain }}.{{ serverStore.capabilities.subdomain_host }}
          </div>
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" @click="handleSubmit">创建</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.hint { font-size: 12px; color: #909399; margin-top: 4px; }
</style>
