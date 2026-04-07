<template>
  <div class="balance-page">
    <el-card>
      <template #header>
        <span>余额管理</span>
      </template>
      
      <div class="balance-info">
        <div class="balance-card">
          <div class="balance-label">当前余额</div>
          <div class="balance-value">{{ balance.quota || 0 }}</div>
          <div class="balance-unit">积分</div>
        </div>
        <div class="balance-stats">
          <div class="stat-item">
            <div class="stat-label">总配额</div>
            <div class="stat-value">{{ balance.total_quota || 0 }}</div>
          </div>
          <div class="stat-item">
            <div class="stat-label">已使用</div>
            <div class="stat-value">{{ (balance.total_quota || 0) - (balance.quota || 0) }}</div>
          </div>
          <div class="stat-item">
            <div class="stat-label">使用率</div>
            <div class="stat-value">{{ usageRate }}%</div>
          </div>
        </div>
      </div>
      
      <el-divider>充值</el-divider>
      
      <div class="recharge-section">
        <el-row :gutter="20">
          <el-col :span="8" v-for="plan in rechargePlans" :key="plan.amount">
            <el-card :body-style="{ padding: '20px' }" @click="selectPlan(plan)">
              <div class="plan-card" :class="{ active: selectedPlan === plan }">
                <div class="plan-amount">{{ plan.amount }}</div>
                <div class="plan-price">{{ plan.price }} 元</div>
                <div class="plan-description">{{ plan.description }}</div>
              </div>
            </el-card>
          </el-col>
        </el-row>
        <div class="recharge-button">
          <el-button type="primary" size="large" @click="recharge" :disabled="!selectedPlan">
            立即充值
          </el-button>
        </div>
      </div>
      
      <el-divider>消费记录</el-divider>
      
      <div class="billing-section">
        <el-row :gutter="20" class="mb-4">
          <el-col :span="12">
            <el-date-picker
              v-model="dateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              @change="loadBillings"
            />
          </el-col>
          <el-col :span="12">
            <el-select v-model="billingType" placeholder="类型" clearable @change="loadBillings">
              <el-option label="全部" value="" />
              <el-option label="消费" value="0" />
              <el-option label="充值" value="1" />
              <el-option label="赠送" value="4" />
            </el-select>
          </el-col>
        </el-row>
        
        <el-table :data="billings" style="width: 100%">
          <el-table-column prop="created_at" label="时间" width="180" />
          <el-table-column prop="model_name" label="模型" />
          <el-table-column prop="total_tokens" label="Token数" width="100" />
          <el-table-column prop="quota" label="消耗积分" width="100" />
          <el-table-column prop="type" label="类型" width="100">
            <template #default="{ row }">
              <el-tag :type="getBillingTypeTag(row.type)">
                {{ getBillingTypeName(row.type) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="channel_name" label="渠道" />
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
      </div>
    </el-card>
  </div>
</template>

<script>
import { computed, onMounted, ref } from 'vue'

export default {
  setup() {
    const balance = ref({})
    const billings = ref([])
    const total = ref(0)
    const currentPage = ref(1)
    const pageSize = ref(20)
    const dateRange = ref([])
    const billingType = ref('')
    const rechargePlans = ref([
      { amount: 1000, price: 10, description: '基础套餐' },
      { amount: 5000, price: 45, description: '进阶套餐' },
      { amount: 10000, price: 80, description: '高级套餐' },
      { amount: 50000, price: 350, description: '企业套餐' }
    ])
    const selectedPlan = ref(null)
    
    const usageRate = computed(() => {
      if (!balance.value.total_quota) return 0
      return Math.round(((balance.value.total_quota - balance.value.quota) / balance.value.total_quota) * 100)
    })
    
    const loadBalance = async () => {
      try {
        const response = await api.getBalance()
        balance.value = response.data
      } catch (error) {
        console.error('加载余额失败', error)
      }
    }
    
    const loadBillings = async () => {
      try {
        const params = {
          page: currentPage.value,
          page_size: pageSize.value
        }
        if (dateRange.value.length === 2) {
          params.start = dateRange.value[0]
          params.end = dateRange.value[1]
        }
        if (billingType.value) {
          params.type = billingType.value
        }
        const response = await api.getBillings(params)
        // 新响应格式: { code, message, data: { items: [...], total: N, page: N, page_size: N } }
        billings.value = response.data.data?.items || []
        total.value = response.data.data?.total || 0
      } catch (error) {
        console.error('加载消费记录失败', error)
      }
    }
    
    const handleSizeChange = (size) => {
      pageSize.value = size
      loadBillings()
    }
    
    const handleCurrentChange = (page) => {
      currentPage.value = page
      loadBillings()
    }
    
    const selectPlan = (plan) => {
      selectedPlan.value = plan
    }
    
    const recharge = () => {
      if (!selectedPlan.value) return
      // 这里可以实现充值逻辑，调用支付API
      alert(`充值 ${selectedPlan.value.amount} 积分，价格 ${selectedPlan.value.price} 元`)
    }
    
    const getBillingTypeTag = (type) => {
      switch (type) {
        case 0: return 'danger'
        case 1: return 'success'
        case 4: return 'warning'
        default: return 'default'
      }
    }
    
    const getBillingTypeName = (type) => {
      switch (type) {
        case 0: return '消费'
        case 1: return '充值'
        case 4: return '赠送'
        default: return '未知'
      }
    }
    
    onMounted(() => {
      loadBalance()
      loadBillings()
    })
    
    return {
      balance,
      billings,
      total,
      currentPage,
      pageSize,
      dateRange,
      billingType,
      rechargePlans,
      selectedPlan,
      usageRate,
      loadBalance,
      loadBillings,
      handleSizeChange,
      handleCurrentChange,
      selectPlan,
      recharge,
      getBillingTypeTag,
      getBillingTypeName
    }
  }
}
</script>

<style scoped>
.balance-page {
  padding: 20px;
}

.balance-info {
  margin-bottom: 30px;
}

.balance-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 30px;
  border-radius: 12px;
  text-align: center;
  margin-bottom: 20px;
}

.balance-label {
  font-size: 16px;
  opacity: 0.8;
  margin-bottom: 10px;
}

.balance-value {
  font-size: 48px;
  font-weight: bold;
  margin-bottom: 10px;
}

.balance-unit {
  font-size: 18px;
  opacity: 0.8;
}

.balance-stats {
  display: flex;
  gap: 20px;
}

.stat-item {
  flex: 1;
  background: #f5f7fa;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
}

.stat-label {
  font-size: 14px;
  color: #606266;
  margin-bottom: 10px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}

.recharge-section {
  margin-bottom: 30px;
}

.plan-card {
  text-align: center;
  padding: 20px;
  border: 2px solid #e4e7ed;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
}

.plan-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.plan-card.active {
  border-color: #409eff;
  background-color: #ecf5ff;
}

.plan-amount {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
  margin-bottom: 10px;
}

.plan-price {
  font-size: 18px;
  color: #409eff;
  margin-bottom: 10px;
}

.plan-description {
  font-size: 14px;
  color: #606266;
}

.recharge-button {
  text-align: center;
  margin-top: 30px;
}

.billing-section {
  margin-top: 30px;
}

.mb-4 {
  margin-bottom: 16px;
}
</style>
