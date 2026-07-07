<script setup lang="ts">
</script>

<template>
  <div class="page">
    <h2>设置</h2>
    <el-form label-width="160px" style="max-width: 520px">
      <el-form-item label="开机自启">
        <el-switch />
      </el-form-item>
      <el-form-item label="关闭时最小化">
        <el-switch />
      </el-form-item>
      <el-form-item label="日志保留">
        <el-input-number :min="1" :max="30" />
        <span class="hint">天</span>
      </el-form-item>
      <el-form-item label="frpc 路径">
        <el-input placeholder="未指定则使用系统 PATH 中的 frpc" />
      </el-form-item>
      <el-form-item>
        <el-button type="primary">保存</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped>
.page { padding: 16px; }
.page h2 { margin: 0 0 16px 0; }
.hint { margin-left: 8px; color: #909399; }
</style>
