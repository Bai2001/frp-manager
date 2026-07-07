<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { type AddServerInput, type ServerInfo } from '@/api'

const props = defineProps<{
  visible: boolean
  editing?: ServerInfo | null
}>()
const emit = defineEmits<{
  (e: 'update:visible', v: boolean): void
  (e: 'submit', input: AddServerInput): void
}>()

const formRef = ref<FormInstance>()
const form = reactive<AddServerInput>({
  name: '', host: '', frps_port: 7000, frp_token: '',
  agent_url: '', agent_token: '', is_default: false, remark: '',
})

const rules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  host: [{ required: true, message: '请输入公网地址', trigger: 'blur' }],
  frps_port: [{ required: true, message: '请输入 frps 端口', trigger: 'blur' }],
  frp_token: [{ required: true, message: '请输入 frp token', trigger: 'blur' }],
  agent_url: [{ required: true, message: '请输入 agent 地址', trigger: 'blur' }],
  agent_token: [{ required: true, message: '请输入 agent token', trigger: 'blur' }],
}

watch(() => props.visible, (v) => {
  if (v) {
    if (props.editing) {
      Object.assign(form, {
        name: props.editing.name, host: props.editing.host,
        frps_port: props.editing.frps_port, frp_token: props.editing.frp_token,
        agent_url: props.editing.agent_url, agent_token: props.editing.agent_token,
        is_default: props.editing.is_default, remark: props.editing.remark ?? '',
      })
    } else {
      Object.assign(form, {
        name: '', host: '', frps_port: 7000, frp_token: '',
        agent_url: '', agent_token: '', is_default: false, remark: '',
      })
    }
  }
})

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate((valid) => {
    if (valid) emit('submit', { ...form })
  })
}

function close() {
  emit('update:visible', false)
}
</script>

<template>
  <el-dialog :model-value="visible" :title="editing ? '编辑服务器' : '添加服务器'" width="520px" @update:model-value="close">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
      <el-form-item label="名称" prop="name"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="公网地址" prop="host"><el-input v-model="form.host" placeholder="IP 或域名" /></el-form-item>
      <el-form-item label="frps 端口" prop="frps_port"><el-input-number v-model="form.frps_port" :min="1" :max="65535" /></el-form-item>
      <el-form-item label="frp token" prop="frp_token"><el-input v-model="form.frp_token" show-password /></el-form-item>
      <el-form-item label="Agent 地址" prop="agent_url"><el-input v-model="form.agent_url" placeholder="http://1.2.3.4:7400" /></el-form-item>
      <el-form-item label="Agent token" prop="agent_token"><el-input v-model="form.agent_token" show-password /></el-form-item>
      <el-form-item label="设为默认"><el-switch v-model="form.is_default" /></el-form-item>
      <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="close">取消</el-button>
      <el-button type="primary" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>
