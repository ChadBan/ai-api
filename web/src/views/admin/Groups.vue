<template>
  <div class="groups-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>分组管理</span>
          <el-button type="primary" @click="showCreateDialog = true">
            <el-icon><Plus /></el-icon>
            新增分组
          </el-button>
        </div>
      </template>
      
      <div class="search-bar">
        <el-input
          v-model="searchQuery"
          placeholder="搜索分组名称"
          prefix-icon="Search"
          @keyup.enter="loadGroups"
        >
          <template #append>
            <el-button @click="loadGroups">
              <el-icon><Search /></el-icon>
            </el-button>
          </template>
        </el-input>
      </div>
      
      <el-table :data="groups" v-loading="loading" border style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="分组名称" />
        <el-table-column prop="display_name" label="显示名称" />
        <el-table-column prop="ratio" label="定价倍率" width="100">
          <template #default="{ row }">
            {{ row.ratio.toFixed(2) }}x
          </template>
        </el-table-column>
        <el-table-column prop="qps_limit" label="QPS限制" width="100" />
        <el-table-column prop="daily_limit" label="每日限制" width="100" />
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="editGroup(row)">
              <el-icon><Edit /></el-icon>
              编辑
            </el-button>
            <el-button 
              size="small" 
              type="danger" 
              @click="deleteGroup(row.id)" 
              :disabled="row.name === 'default'"
            >
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    
    <!-- 新增分组对话框 -->
    <el-dialog v-model="showCreateDialog" title="新增分组" width="600px">
      <el-form :model="createForm" label-width="120px">
        <el-form-item label="分组名称" required>
          <el-input v-model="createForm.name" placeholder="请输入分组名称" />
        </el-form-item>
        <el-form-item label="显示名称" required>
          <el-input v-model="createForm.display_name" placeholder="请输入显示名称" />
        </el-form-item>
        <el-form-item label="定价倍率">
          <el-input-number v-model="createForm.ratio" :min="0.1" :step="0.1" :precision="2" />
        </el-form-item>
        <el-form-item label="QPS限制">
          <el-input-number v-model="createForm.qps_limit" :min="0" :step="1" />
          <span class="form-hint">0表示使用系统默认</span>
        </el-form-item>
        <el-form-item label="每日限制">
          <el-input-number v-model="createForm.daily_limit" :min="0" :step="100" />
          <span class="form-hint">0表示无限制</span>
        </el-form-item>
        <el-form-item label="允许模型">
          <el-input v-model="createForm.models" type="textarea" placeholder="JSON格式，如 ['gpt-3.5-turbo', 'gpt-4']，空表示允许所有模型" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="createForm.status" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" placeholder="请输入分组描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetCreateForm">取消</el-button>
        <el-button type="primary" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
    
    <!-- 编辑分组对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑分组" width="600px">
      <el-form :model="editForm" label-width="120px">
        <el-form-item label="分组名称" required :disabled="editForm.name === 'default'">
          <el-input v-model="editForm.name" :disabled="editForm.name === 'default'" />
        </el-form-item>
        <el-form-item label="显示名称" required>
          <el-input v-model="editForm.display_name" />
        </el-form-item>
        <el-form-item label="定价倍率">
          <el-input-number v-model="editForm.ratio" :min="0.1" :step="0.1" :precision="2" />
        </el-form-item>
        <el-form-item label="QPS限制">
          <el-input-number v-model="editForm.qps_limit" :min="0" :step="1" />
          <span class="form-hint">0表示使用系统默认</span>
        </el-form-item>
        <el-form-item label="每日限制">
          <el-input-number v-model="editForm.daily_limit" :min="0" :step="100" />
          <span class="form-hint">0表示无限制</span>
        </el-form-item>
        <el-form-item label="允许模型">
          <el-input v-model="editForm.models" type="textarea" placeholder="JSON格式，如 ['gpt-3.5-turbo', 'gpt-4']，空表示允许所有模型" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="editForm.status" :disabled="editForm.name === 'default'" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="editForm.description" type="textarea" placeholder="请输入分组描述" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="handleEdit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import api from '@/api';
import { Plus, Edit, Delete, Search } from '@element-plus/icons-vue';
import { ElMessage } from 'element-plus';
import { onMounted, reactive, ref } from 'vue';

const loading = ref(false)
const groups = ref([])
const total = ref(0)
const searchQuery = ref('')
const showCreateDialog = ref(false)
const showEditDialog = ref(false)

const pagination = reactive({
  current: 1,
  pageSize: 20
})

const createForm = reactive({
  name: '',
  display_name: '',
  ratio: 1.0,
  qps_limit: 0,
  daily_limit: 0,
  models: '',
  status: true,
  description: ''
})

const editForm = reactive({
  id: '',
  name: '',
  display_name: '',
  ratio: 1.0,
  qps_limit: 0,
  daily_limit: 0,
  models: '',
  status: true,
  description: ''
})

const loadGroups = async () => {
  loading.value = true
  try {
    const res = await api.admin.getGroups({
      page: pagination.current,
      page_size: pagination.pageSize,
      search: searchQuery.value
    })
    groups.value = res.data.data?.data || []
    total.value = res.data.data?.total || 0
  } catch (error) {
    console.error('Failed to load groups:', error)
    ElMessage.error('加载分组失败')
  } finally {
    loading.value = false
  }
}

const handleSizeChange = (size) => {
  pagination.pageSize = size
  loadGroups()
}

const handleCurrentChange = (current) => {
  pagination.current = current
  loadGroups()
}

const resetCreateForm = () => {
  createForm.name = ''
  createForm.display_name = ''
  createForm.ratio = 1.0
  createForm.qps_limit = 0
  createForm.daily_limit = 0
  createForm.models = ''
  createForm.status = true
  createForm.description = ''
  showCreateDialog.value = false
}

const handleCreate = async () => {
  try {
    const data = {
      ...createForm,
      status: createForm.status ? 1 : 0
    }
    await api.admin.createGroup(data)
    ElMessage.success('创建成功')
    resetCreateForm()
    loadGroups()
  } catch (error) {
    console.error('Failed to create group:', error)
    ElMessage.error('创建失败')
  }
}

const editGroup = (group) => {
  editForm.id = group.id
  editForm.name = group.name
  editForm.display_name = group.display_name
  editForm.ratio = group.ratio
  editForm.qps_limit = group.qps_limit
  editForm.daily_limit = group.daily_limit
  editForm.models = group.models
  editForm.status = group.status === 1
  editForm.description = group.description
  showEditDialog.value = true
}

const handleEdit = async () => {
  try {
    const data = {
      ...editForm,
      status: editForm.status ? 1 : 0
    }
    await api.admin.updateGroup(editForm.id, data)
    ElMessage.success('更新成功')
    showEditDialog.value = false
    loadGroups()
  } catch (error) {
    console.error('Failed to update group:', error)
    ElMessage.error('更新失败')
  }
}

const deleteGroup = async (id) => {
  if (!confirm('确定要删除此分组吗？')) return
  try {
    await api.admin.deleteGroup(id)
    ElMessage.success('删除成功')
    loadGroups()
  } catch (error) {
    console.error('Failed to delete group:', error)
    ElMessage.error('删除失败')
  }
}

onMounted(() => {
  loadGroups()
})
</script>

<style scoped lang="scss">
.groups-page {
  padding: 20px;
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .search-bar {
    margin-bottom: 20px;
  }
  
  .pagination {
    margin-top: 20px;
    display: flex;
    justify-content: flex-end;
  }
  
  .form-hint {
    margin-left: 10px;
    color: #909399;
    font-size: 12px;
  }
  
  .el-table {
    .el-table__row {
      transition: all 0.3s ease;
      
      &:hover {
        background-color: #f5f7fa;
      }
    }
  }
}
</style>