<template>
  <div>
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>用户总数</template>
          <div class="stat-value">--</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>角色数量</template>
          <div class="stat-value">--</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>菜单数量</template>
          <div class="stat-value">--</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>API 数量</template>
          <div class="stat-value">--</div>
        </el-card>
      </el-col>
    </el-row>
    <el-card>
      <template #header>欢迎使用 Backplane Admin</template>
      <p>这是一个基于 <strong>Go (Gin + GORM + Casbin)</strong> 和 <strong>Vue 3 (Element Plus)</strong> 构建的后台管理框架。</p>
      <p>主要功能包括：</p>
      <ul>
        <li>JWT 鉴权</li>
        <li>基于 Casbin 的 RBAC 权限管理</li>
        <li>用户 / 角色 / 菜单 管理</li>
        <li>模块化路由架构</li>
      </ul>
    </el-card>
  </div>
</template>

<style scoped>
.stat-value {
  font-size: 32px;
  font-weight: bold;
  color: #409eff;
  text-align: center;
}
</style>
