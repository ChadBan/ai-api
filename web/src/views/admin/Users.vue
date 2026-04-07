<template>
  <div class="users-page">
    <el-card>
      <template #header>
        <span>用户管理</span>
      </template>
      
      <el-row :gutter="20" class="mb-4">
        <el-col :span="8">
          <el-input
            v-model="searchQuery"
            placeholder="搜索用户"
            clearable
            @keyup.enter="loadUsers"
          >
            <template #append>
              <el-button @click="loadUsers"><el-icon><Search /></el-icon></el-button>
            </template>
          </el-input>
        </el-col>
        <el-col :span="8">
          <el-select v-model="statusFilter" placeholder="状态" clearable @change="loadUsers">
            <el-option label="正常" value="1" />
            <el-option label="封禁" value="0" />
          </el-select>
        </el-col>
        <el-col :span="8">
          <el-select v-model="roleFilter" placeholder="角色" clearable @change="loadUsers">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-col>
      </el-row>
      
      <el-table :data="users" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="email" label="邮箱" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="role" label="角色" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.role === 'admin' ? 'warning' : 'info'">
              {{ scope.row.role === 'admin' ? '管理员' : '用户' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 1 ? 'success' : 'danger'">
              {{ scope.row.status === 1 ? '正常' : '封禁' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tier" label="等级" width="100" />
        <el-table-column prop="created_at" label="创建时间" width="180" />
        <el-table-column label="操作" width="200">
          <template #default="scope">
            <el-button size="small" @click="editUser(scope.row)">编辑</el-button>
            <el-button 
              size="small" 
              :type="scope.row.status === 1 ? 'danger' : 'success'"
              @click="toggleBan(scope.row)"
            >
              {{ scope.row.status === 1 ? '封禁' : '解封' }}
            </el-button>
            <el-button size="small" type="primary" @click="addBalance(scope.row)">加余额</el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
        class="mt-4"
      />
    </el-card>
    
    <!-- 编辑用户对话框 -->
    <el-dialog v-model="editDialogVisible" title="编辑用户">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="editForm.role">
            <el-option label="普通用户" value="user" />
            <el-option label="管理员" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item label="等级">
          <el-select v-model="editForm.tier">
            <el-option label="免费" value="free" />
            <el-option label="专业" value="pro" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="updateUser">确定</el-button>
        </span>
      </template>
    </el-dialog>
    
    <!-- 加余额对话框 -->
    <el-dialog v-model="balanceDialogVisible" title="添加余额">
      <el-form :model="balanceForm" label-width="80px">
        <el-form-item label="用户">
          <el-input v-model="balanceForm.userEmail" disabled />
        </el-form-item>
        <el-form-item label="配额" required>
          <el-input-number v-model="balanceForm.quota" :min="1" :step="1000" />
        </el-form-item>
        <el-form-item label="原因">
          <el-input v-model="balanceForm.reason" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="balanceDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitBalance">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { Search } from '@element-plus/icons-vue'

export default {
  components: {
    Search
  },
  data() {
    return {
      users: [],
      total: 0,
      currentPage: 1,
      pageSize: 20,
      searchQuery: '',
      statusFilter: '',
      roleFilter: '',
      editDialogVisible: false,
      balanceDialogVisible: false,
      editForm: {
        id: '',
        name: '',
        role: '',
        tier: ''
      },
      balanceForm: {
        userId: '',
        userEmail: '',
        quota: 1000,
        reason: ''
      }
    }
  },
  mounted() {
    this.loadUsers()
  },
  methods: {
    async loadUsers() {
      try {
        const response = await this.$api.admin.getUsers({
          page: this.currentPage,
          page_size: this.pageSize,
          status: this.statusFilter,
          role: this.roleFilter
        })
        // 新响应格式: { code, message, data: { items: [...], total: N, page: N, page_size: N } }
        this.users = response.data.data?.items || []
        this.total = response.data.data?.total || 0
      } catch (error) {
        this.$message.error('加载用户失败')
      }
    },
    handleSizeChange(size) {
      this.pageSize = size
      this.loadUsers()
    },
    handleCurrentChange(page) {
      this.currentPage = page
      this.loadUsers()
    },
    editUser(user) {
      this.editForm = {
        id: user.id,
        name: user.name,
        role: user.role,
        tier: user.tier
      }
      this.editDialogVisible = true
    },
    async updateUser() {
      try {
        await this.$api.admin.updateUser(this.editForm.id, {
          name: this.editForm.name,
          role: this.editForm.role,
          tier: this.editForm.tier
        })
        this.$message.success('更新成功')
        this.editDialogVisible = false
        this.loadUsers()
      } catch (error) {
        this.$message.error('更新失败')
      }
    },
    async toggleBan(user) {
      try {
        await this.$api.admin.banUser(user.id, { ban: user.status === 1 })
        this.$message.success(user.status === 1 ? '封禁成功' : '解封成功')
        this.loadUsers()
      } catch (error) {
        this.$message.error('操作失败')
      }
    },
    addBalance(user) {
      this.balanceForm = {
        userId: user.id,
        userEmail: user.email,
        quota: 1000,
        reason: ''
      }
      this.balanceDialogVisible = true
    },
    async submitBalance() {
      try {
        await this.$api.admin.addBalance(this.balanceForm.userId, {
          quota: this.balanceForm.quota,
          reason: this.balanceForm.reason
        })
        this.$message.success('添加余额成功')
        this.balanceDialogVisible = false
      } catch (error) {
        this.$message.error('添加余额失败')
      }
    }
  }
}
</script>

<style scoped>
.users-page {
  padding: 20px;
}

.mb-4 {
  margin-bottom: 16px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
