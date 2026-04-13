<template>
  <div class="dashboard">
    <el-row :gutter="20">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #409EFF;">
              <el-icon><Wallet /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ balance?.quota || 0 }}</div>
              <div class="stat-label">可用积分</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #67C23A;">
              <el-icon><DataLine /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats?.total_requests || 0 }}</div>
              <div class="stat-label">今日请求</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #E6A23C;">
              <el-icon><TrendCharts /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats?.total_tokens || 0 }}</div>
              <div class="stat-label">总 Token 消耗</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" style="background: #F56C6C;">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats?.active_tokens || 0 }}</div>
              <div class="stat-label">活跃 Token</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>快速操作</span>
            </div>
          </template>
          
          <div class="quick-actions">
            <el-button type="primary" @click="$router.push('/tokens')">
              <el-icon><Plus /></el-icon>
              创建 Token
            </el-button>
            <el-button type="success" @click="$router.push('/balance')">
              <el-icon><Coin /></el-icon>
              充值余额
            </el-button>
            <el-button type="warning" @click="$router.push('/playground')">
              <el-icon><ChatDotRound /></el-icon>
              AI 对话
            </el-button>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近公告</span>
            </div>
          </template>
          
          <el-empty description="暂无公告" />
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>请求趋势</span>
              <el-select v-model="timeRange" placeholder="时间范围" size="small" @change="loadChartData">
                <el-option label="今日" value="today" />
                <el-option label="近7天" value="7days" />
                <el-option label="近30天" value="30days" />
              </el-select>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!chartData.length" description="暂无数据" />
            <v-chart v-else :option="requestChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>模型使用分布</span>
            </div>
          </template>
          <div class="chart-container">
            <el-empty v-if="!modelData.length" description="暂无数据" />
            <v-chart v-else :option="modelChartOption" style="width: 100%; height: 300px;" />
          </div>
        </el-card>
      </el-col>
    </el-row>
    
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>最近请求</span>
              <el-button type="primary" size="small" @click="refreshRequests">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          <el-table :data="recentRequests" style="width: 100%" stripe>
            <el-table-column prop="id" label="ID" width="80" />
            <el-table-column prop="model_name" label="模型" />
            <el-table-column prop="path" label="路径" />
            <el-table-column prop="status_code" label="状态码" width="100">
              <template #default="scope">
                <el-tag :type="scope.row.status_code === 200 ? 'success' : 'danger'">
                  {{ scope.row.status_code }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="duration" label="响应时间(ms)" width="120" />
            <el-table-column prop="created_at" label="时间" width="180" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import api from '@/api'
import { useUserStore } from '@/store/user'
import { ChatDotRound, Coin, DataLine, Key, Plus, Refresh, TrendCharts, Wallet } from '@element-plus/icons-vue'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import {
GridComponent,
LegendComponent,
TitleComponent,
ToolboxComponent,
TooltipComponent
} from 'echarts/components'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { computed, onMounted, ref } from 'vue'
import VChart from 'vue-echarts'

use([
  CanvasRenderer,
  LineChart,
  PieChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  ToolboxComponent
])

const userStore = useUserStore()
const balance = ref(null)
const stats = ref({})
const chartData = ref([])
const modelData = ref([])
const recentRequests = ref([])
const timeRange = ref('today')

const requestChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: {
          backgroundColor: '#6a7985'
        }
      }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: chartData.value.map(item => item.time)
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '请求数',
        type: 'line',
        stack: 'Total',
        areaStyle: {
          opacity: 0.3
        },
        emphasis: {
          focus: 'series'
        },
        data: chartData.value.map(item => item.value),
        lineStyle: {
          color: '#409EFF'
        },
        itemStyle: {
          color: '#409EFF'
        }
      }
    ]
  }
})

const modelChartOption = computed(() => {
  return {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '模型使用',
        type: 'pie',
        radius: '50%',
        data: modelData.value.map(item => ({ name: item.model, value: item.count })),
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
})

const loadChartData = async () => {
  try {
    // 从API获取真实数据
    const res = await api.getChartData(timeRange.value)
    const data = res.data?.data || res.data || {}
    chartData.value = data.request_trend || []
    modelData.value = data.model_usage || []
  } catch (error) {
    console.error('加载图表数据失败:', error)
    chartData.value = []
    modelData.value = []
  }
}

const loadRecentRequests = async () => {
  try {
    // 从API获取真实数据
    const res = await api.getRecentRequests()
    const data = res.data?.data || res.data || {}
    recentRequests.value = data.requests || []
  } catch (error) {
    console.error('加载最近请求失败:', error)
    recentRequests.value = []
  }
}

const refreshRequests = async () => {
  await loadRecentRequests()
}

onMounted(async () => {
  try {
    const [balanceRes, statsRes, chartRes, requestsRes] = await Promise.all([
      api.getBalance(),
      api.getUserDashboard(),
      api.getChartData(timeRange.value),
      api.getRecentRequests()
    ])
    
    // 新响应格式: { code, message, data: {...} }
    balance.value = balanceRes.data?.data || balanceRes.data
    stats.value = statsRes.data?.data || statsRes.data || {}
    
    // 处理图表数据
    const chartDataRes = chartRes.data?.data || chartRes.data || {}
    chartData.value = chartDataRes.request_trend || []
    modelData.value = chartDataRes.model_usage || []
    
    // 处理最近请求
    const requestsData = requestsRes.data?.data || requestsRes.data || {}
    recentRequests.value = requestsData.requests || []
  } catch (error) {
    console.error('Failed to load dashboard data:', error)
  }
})
</script>

<style scoped lang="scss">
.dashboard {
  .stat-card {
    transition: all 0.3s ease;
    
    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    }
    
    .stat-content {
      display: flex;
      align-items: center;
      gap: 16px;
      
      .stat-icon {
        width: 60px;
        height: 60px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        
        .el-icon {
          font-size: 28px;
          color: #fff;
        }
      }
      
      .stat-info {
        flex: 1;
        
        .stat-value {
          font-size: 24px;
          font-weight: 600;
          color: #303133;
        }
        
        .stat-label {
          font-size: 14px;
          color: #909399;
          margin-top: 4px;
        }
      }
    }
  }
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    
    span {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }
  }
  
  .quick-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }
  
  .chart-container {
    padding: 20px;
    height: 300px;
  }
  
  .el-table {
    margin-top: 10px;
    
    .el-table__row {
      transition: all 0.3s ease;
      
      &:hover {
        background-color: #f5f7fa;
      }
    }
  }
}
</style>
